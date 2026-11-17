import { afterEach, expect, it, vi } from 'vitest';
import { createUppy } from './uppy';
import { createUppyEngine } from './engine';
import { createUploadQueue } from './queue';

const instances: ReturnType<typeof createUppy>[] = [];
afterEach(() => instances.splice(0).forEach(uppy => uppy.destroy()));

function fixture(hashFile = async (_file: File) => 'a'.repeat(64)) {
  const uppy = createUppy();
  instances.push(uppy);
  const engine = createUppyEngine(uppy);
  // Exercise the real identity and event adapter without making network requests.
  const queue = createUploadQueue({ engine: { ...engine, start() {} }, hashFile });
  return { uppy, engine, queue };
}

it('retains independent tasks and destinations for the same file', async () => {
  const { uppy, queue } = fixture();
  const file = new File(['{}'], 'same.json', { lastModified: 1 });
  await queue.enqueue([file], 'helpdoc', { group: 'a' });
  await queue.enqueue([file], 'helpdoc', { group: 'b' });
  const uploads = uppy.getFiles();
  expect(new Set(uploads.map(item => item.id)).size).toBe(2);
  expect(uploads.map(item => item.meta.group)).toEqual(['a', 'b']);
  expect(uploads.map(item => item.meta.taskId)).toEqual(queue.tasks.value.map(item => item.id));
  queue.cancel(queue.tasks.value[0]!.id);
  expect(uppy.getFiles().map(item => item.meta.group)).toEqual(['b']);
});

it('rejects an active duplicate without overwriting the original task', async () => {
  const { uppy, queue } = fixture();
  const file = new File(['{}'], 'same.json', { lastModified: 1 });
  await queue.enqueue([file, file], 'helpdoc', { group: 'a' });
  expect(uppy.getFiles()).toHaveLength(1);
  expect(uppy.getFiles()[0]!.meta.taskId).toBe(queue.tasks.value[0]!.id);
  expect(queue.tasks.value[1]!.status).toBe('error');
});

it('settles an awaiting installation when the task is canceled', async () => {
  const { queue } = fixture();
  const pending = queue.upload(new File(['x'], 'package.seal'), 'extension-install');
  const rejected = expect(pending).rejects.toMatchObject({ name: 'AbortError' });
  queue.cancelAll();
  await rejected;
  expect(queue.hasActive.value).toBe(false);
});

it('releases a completed file so the same destination can be uploaded again', async () => {
  const { uppy, queue } = fixture();
  const file = new File(['{}'], 'same.json', { lastModified: 1 });
  await queue.enqueue([file], 'helpdoc', { group: 'a' });
  const uploaded = uppy.getFiles()[0]!;
  uppy.emit('upload-success', uploaded, { status: 200, body: {}, uploadURL: 'test' });
  await Promise.resolve();
  expect(queue.tasks.value).toHaveLength(0);
  expect(uppy.getFiles()).toHaveLength(0);
  await queue.enqueue([file], 'helpdoc', { group: 'a' });
  expect(uppy.getFiles()).toHaveLength(1);
});

it('retries preparation after a file read failure instead of leaving a phantom task', async () => {
  const hash = vi
    .fn()
    .mockRejectedValueOnce(new Error('read failed'))
    .mockResolvedValue('a'.repeat(64));
  const { uppy, queue } = fixture(hash);
  await queue.enqueue([new File(['x'], 'a.png')], 'resource');
  expect(queue.hasActive.value).toBe(false);
  await queue.retry(queue.tasks.value[0]!.id);
  expect(hash).toHaveBeenCalledTimes(2);
  expect(uppy.getFiles()).toHaveLength(1);
  expect(queue.tasks.value[0]!.status).toBe('queued');
});

it('keeps a repeated preparation failure terminal and does not retry invalid types', async () => {
  const hash = vi.fn().mockRejectedValue(new Error('read failed'));
  const { uppy, queue } = fixture(hash);
  await queue.enqueue([new File(['x'], 'a.png'), new File(['x'], 'a.txt')], 'resource');
  for (const task of queue.tasks.value) await queue.retry(task.id);
  expect(queue.tasks.value.every(task => task.status === 'error')).toBe(true);
  expect(queue.tasks.value[1]!.retryable).toBe(false);
  expect(queue.hasActive.value).toBe(false);
  expect(uppy.getFiles()).toHaveLength(0);
  expect(hash).toHaveBeenCalledTimes(2);
});

it('cancels the entire batch during hashing and settles cancellation callbacks', async () => {
  let resolveHash!: (hash: string) => void;
  const { uppy, queue } = fixture(
    () =>
      new Promise(resolve => {
        resolveHash = resolve;
      })
  );
  const onCancel = vi.fn();
  const pending = queue.enqueue([new File(['x'], 'a.png'), new File(['y'], 'b.png')], 'resource', {
    onCancel,
  });
  queue.cancelAll();
  resolveHash('a'.repeat(64));
  await pending;
  expect(onCancel).toHaveBeenCalledTimes(2);
  expect(queue.hasActive.value).toBe(false);
  expect(uppy.getFiles()).toHaveLength(0);
});
