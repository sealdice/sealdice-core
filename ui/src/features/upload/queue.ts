import { computed, ref } from 'vue';
import { isFileAllowed, UPLOAD_KINDS, type UploadKindContext, type UploadKindID } from './kinds';
import type { UploadEngine } from './engine';

export type UploadTaskStatus =
  | 'queued'
  | 'hashing'
  | 'uploading'
  | 'success'
  | 'error'
  | 'canceled';
export type UploadTask = {
  id: string;
  kind: UploadKindID;
  filename: string;
  size: number;
  sha256: string;
  status: UploadTaskStatus;
  progress: number;
  uploadedBytes: number;
  errorText: string;
  retryable: boolean;
};
export type EnqueueContext = UploadKindContext & {
  onComplete?: (task: UploadTask) => void;
  onError?: (task: UploadTask, error: Error) => void;
  onCancel?: (task: UploadTask) => void;
};
export type UploadQueue = ReturnType<typeof createUploadQueue>;
export const isActiveUpload = (status: UploadTaskStatus) =>
  status === 'queued' || status === 'hashing' || status === 'uploading';

export function createUploadQueue(deps: {
  engine: UploadEngine;
  hashFile: (file: File) => Promise<string>;
  newId?: () => string;
}) {
  const tasks = ref<UploadTask[]>([]);
  // Retain inputs until completion/cancellation so preparation failures can be retried.
  const inputs = new Map<string, { file: File; context: EnqueueContext }>();
  let seq = 0;
  const newId = deps.newId ?? (() => `up-${Date.now()}-${(seq += 1)}`);
  const find = (id: string) => tasks.value.find(task => task.id === id);
  const remove = (id: string) => {
    tasks.value = tasks.value.filter(task => task.id !== id);
    inputs.delete(id);
  };
  function fail(id: string, error: unknown, body?: string) {
    const task = find(id);
    if (!task) return;
    const reason = error instanceof Error ? error : new Error(String(error));
    task.status = 'error';
    task.errorText = body?.trim() || reason.message || '上传失败';
    inputs.get(id)?.context.onError?.(task, reason);
  }

  deps.engine.setHandlers({
    onProgress(id, bytesSent, bytesTotal) {
      const task = find(id);
      if (!task || !isActiveUpload(task.status)) return;
      task.status = 'uploading';
      task.uploadedBytes = bytesSent;
      task.progress =
        bytesTotal > 0 ? Math.min(100, Math.floor((bytesSent / bytesTotal) * 100)) : 0;
    },
    onSuccess(id) {
      const task = find(id);
      if (!task) return;
      task.status = 'success';
      task.progress = 100;
      task.uploadedBytes = task.size;
      const context = inputs.get(id)?.context;
      remove(id);
      context?.onComplete?.(task);
    },
    onError: (id, error, details) => fail(id, error, details?.body),
  });

  async function prepare(id: string) {
    const task = find(id);
    const input = inputs.get(id);
    if (!task || !input) return;
    task.status = 'hashing';
    task.errorText = '';
    task.progress = 0;
    try {
      const sha256 = task.sha256 || (await deps.hashFile(input.file));
      if (!find(id)) return; // Cancelled while reading the file.
      task.sha256 = sha256;
      const def = UPLOAD_KINDS[task.kind];
      deps.engine.add(input.file, {
        taskId: id,
        module: def.module,
        filename: task.filename,
        sha256,
        ...def.extraMeta?.(input.context),
      });
      task.status = 'queued';
    } catch (error) {
      fail(id, error);
    }
  }

  async function enqueue(files: File[], kind: UploadKindID, context: EnqueueContext = {}) {
    // Register the entire batch first: cancelAll must include files not yet hashed.
    const pending = files.map(file => {
      const task: UploadTask = {
        id: newId(),
        kind,
        filename: file.name,
        size: file.size,
        sha256: '',
        status: 'queued',
        progress: 0,
        uploadedBytes: 0,
        errorText: '',
        retryable: isFileAllowed(kind, file.name),
      };
      tasks.value.push(task);
      inputs.set(task.id, { file, context });
      if (!task.retryable) fail(task.id, new Error(`${UPLOAD_KINDS[kind].label}不支持该文件类型`));
      return task;
    });
    for (const task of pending) {
      if (task.retryable) await prepare(task.id);
    }
    deps.engine.start();
  }

  function cancel(id: string) {
    const task = find(id);
    if (!task) return;
    task.status = 'canceled';
    const context = inputs.get(id)?.context;
    // Remove first so any late engine events cannot resurrect the task.
    remove(id);
    deps.engine.cancel(id);
    context?.onCancel?.(task);
  }

  async function retry(id: string) {
    const task = find(id);
    if (!task || task.status !== 'error' || !task.retryable) return;
    task.status = 'queued';
    task.errorText = '';
    task.progress = 0;
    try {
      if (!deps.engine.retry(id)) {
        await prepare(id);
        deps.engine.start();
      }
    } catch (error) {
      fail(id, error);
    }
  }

  function cancelAll() {
    for (const task of tasks.value) {
      if (isActiveUpload(task.status)) cancel(task.id);
    }
  }

  // Unlike enqueue(), this resolves only after installation; cancellation always settles it.
  function upload(file: File, kind: UploadKindID, context: UploadKindContext = {}): Promise<void> {
    return new Promise((resolve, reject) => {
      void enqueue([file], kind, {
        ...context,
        onComplete: () => resolve(),
        onError: (_task, error) => reject(error),
        onCancel: () => reject(new DOMException('已取消上传', 'AbortError')),
      }).catch(reject);
    });
  }

  const hasActive = computed(() => tasks.value.some(task => isActiveUpload(task.status)));
  return { tasks, hasActive, enqueue, upload, cancel, retry, cancelAll };
}
