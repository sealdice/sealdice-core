import { describe, expect, it } from 'vitest';
import { createUploadQueue } from './queue';
import type { UploadEngine, UploadEngineHandlers } from './engine';

type FakeEngine = UploadEngine & {
  added: Array<{ name: string; meta: Record<string, string> }>;
  cancelled: string[];
  retried: string[];
  emitProgress: (id: string, sent: number, total: number) => void;
  emitSuccess: (id: string) => void;
  emitError: (id: string, message: string) => void;
};

function fakeEngine(): FakeEngine {
  let handlers: UploadEngineHandlers | null = null;
  const engine: FakeEngine = {
    added: [],
    cancelled: [],
    retried: [],
    setHandlers(h) {
      handlers = h;
    },
    add(file, meta) {
      engine.added.push({ name: file.name, meta });
    },
    start() {},
    cancel(id) {
      engine.cancelled.push(id);
    },
    retry(id) {
      engine.retried.push(id);
    },
    emitProgress(id, sent, total) {
      handlers?.onProgress(id, sent, total);
    },
    emitSuccess(id) {
      handlers?.onSuccess(id);
    },
    emitError(id, message) {
      handlers?.onError(id, new Error(message));
    },
  };
  return engine;
}

function fileOf(name: string, size = 3): File {
  return new File([new Uint8Array(size)], name);
}

function makeQueue() {
  const engine = fakeEngine();
  let counter = 0;
  const queue = createUploadQueue({
    engine,
    hashFile: async () => 'h'.repeat(64),
    newId: () => `t${(counter += 1)}`,
  });
  return { engine, queue };
}

describe('upload queue', () => {
  it('enqueues a file with module + taskId metadata and starts the engine', async () => {
    const { engine, queue } = makeQueue();
    await queue.enqueue([fileOf('a.png')], 'resource');

    expect(engine.added).toHaveLength(1);
    expect(engine.added[0]!.meta).toMatchObject({
      module: 'resource',
      filename: 'a.png',
      sha256: 'h'.repeat(64),
      taskId: 't1',
    });
    expect(queue.hasActive.value).toBe(true);
  });

  it('reflects progress from the engine', async () => {
    const { engine, queue } = makeQueue();
    await queue.enqueue([fileOf('a.png', 100)], 'resource');
    engine.emitProgress('t1', 50, 100);
    expect(queue.tasks.value[0]!.progress).toBe(50);
  });

  it('runs onComplete and removes the task on success', async () => {
    const { engine, queue } = makeQueue();
    const done: string[] = [];
    await queue.enqueue([fileOf('a.png')], 'resource', {
      onComplete: task => done.push(task.filename),
    });
    engine.emitSuccess('t1');
    expect(done).toEqual(['a.png']);
    expect(queue.tasks.value).toHaveLength(0);
    expect(queue.hasActive.value).toBe(false);
  });

  it('keeps failed tasks and retries them', async () => {
    const { engine, queue } = makeQueue();
    await queue.enqueue([fileOf('a.png')], 'resource');
    engine.emitError('t1', 'boom');
    expect(queue.tasks.value[0]!.status).toBe('error');
    expect(queue.tasks.value[0]!.errorText).toBe('boom');

    queue.retry('t1');
    expect(engine.retried).toEqual(['t1']);
    expect(queue.tasks.value[0]!.status).toBe('queued');
  });

  it('cancels a task through the engine and drops it', async () => {
    const { engine, queue } = makeQueue();
    await queue.enqueue([fileOf('a.png')], 'resource');
    queue.cancel('t1');
    expect(engine.cancelled).toEqual(['t1']);
    expect(queue.tasks.value).toHaveLength(0);
  });

  it('records an error task for a disallowed file type without calling the engine', async () => {
    const { engine, queue } = makeQueue();
    await queue.enqueue([fileOf('a.txt')], 'resource');
    expect(engine.added).toHaveLength(0);
    expect(queue.tasks.value[0]!.status).toBe('error');
  });
});
