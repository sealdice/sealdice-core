import { computed, ref, type ComputedRef, type Ref } from 'vue';
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
};

export type EnqueueContext = UploadKindContext & {
  onComplete?: (task: UploadTask) => void;
  onError?: (task: UploadTask, error: Error) => void;
};

export type UploadQueue = {
  tasks: Ref<UploadTask[]>;
  hasActive: ComputedRef<boolean>;
  enqueue: (files: File[], kind: UploadKindID, ctx?: EnqueueContext) => Promise<void>;
  cancel: (id: string) => void;
  retry: (id: string) => void;
  cancelAll: () => void;
  clearCompleted: () => void;
};

const ACTIVE_STATUSES: UploadTaskStatus[] = ['queued', 'hashing', 'uploading'];

// createUploadQueue 是队列的纯逻辑（依赖注入的引擎 + 哈希），便于无 DOM 测试。
// Pinia store 只是它的薄包装。
export function createUploadQueue(deps: {
  engine: UploadEngine;
  hashFile: (file: File) => Promise<string>;
  newId?: () => string;
}): UploadQueue {
  const tasks = ref<UploadTask[]>([]);
  const contexts = new Map<string, EnqueueContext>();
  let seq = 0;
  const newId = deps.newId ?? (() => `up-${Date.now()}-${(seq += 1)}`);
  const find = (id: string) => tasks.value.find(task => task.id === id);

  // push 后返回数组里的响应式代理：直接改 push 进去的原对象不触发视图更新。
  function push(task: UploadTask): UploadTask {
    tasks.value.push(task);
    return tasks.value[tasks.value.length - 1]!;
  }

  deps.engine.setHandlers({
    onProgress(id, bytesSent, bytesTotal) {
      const task = find(id);
      if (!task) return;
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
      const ctx = contexts.get(id);
      tasks.value = tasks.value.filter(item => item.id !== id);
      contexts.delete(id);
      ctx?.onComplete?.(task);
    },
    onError(id, error, details) {
      const task = find(id);
      if (!task || task.status === 'canceled') return;
      task.status = 'error';
      // 后端失败原因在响应体里（tus 的 error.message 只是通用文案）。
      const backendText = details?.body?.trim();
      task.errorText = backendText || error.message || '上传失败';
      contexts.get(id)?.onError?.(task, error);
    },
  });

  async function enqueue(files: File[], kind: UploadKindID, ctx: EnqueueContext = {}) {
    const def = UPLOAD_KINDS[kind];
    for (const file of files) {
      if (!isFileAllowed(kind, file.name)) {
        push({
          id: newId(),
          kind,
          filename: file.name,
          size: file.size,
          sha256: '',
          status: 'error',
          progress: 0,
          uploadedBytes: 0,
          errorText: `${def.label}不支持该文件类型`,
        });
        continue;
      }
      const task = push({
        id: newId(),
        kind,
        filename: file.name,
        size: file.size,
        sha256: '',
        status: 'hashing',
        progress: 0,
        uploadedBytes: 0,
        errorText: '',
      });
      contexts.set(task.id, ctx);
      try {
        const sha256 = await deps.hashFile(file);
        const current = find(task.id);
        if (!current) continue; // 计算哈希期间被取消
        current.sha256 = sha256;
        current.status = 'queued';
        deps.engine.add(file, {
          taskId: task.id,
          module: def.module,
          filename: file.name,
          sha256,
          ...(def.extraMeta ? def.extraMeta(ctx) : {}),
        });
      } catch (error) {
        // 单个文件入队失败不能中断整批，也不能让调用方的 Promise 永久挂起。
        const current = find(task.id);
        if (!current) continue;
        current.status = 'error';
        const text = error instanceof Error ? error.message : '上传失败';
        current.errorText = text;
        ctx.onError?.(current, error instanceof Error ? error : new Error(text));
      }
    }
    deps.engine.start();
  }

  function cancel(id: string) {
    const task = find(id);
    if (task) task.status = 'canceled';
    deps.engine.cancel(id);
    tasks.value = tasks.value.filter(item => item.id !== id);
    contexts.delete(id);
  }

  function retry(id: string) {
    const task = find(id);
    if (!task || task.status !== 'error') return;
    task.status = 'queued';
    task.errorText = '';
    task.progress = 0;
    deps.engine.retry(id);
  }

  function cancelAll() {
    for (const task of [...tasks.value]) {
      if (ACTIVE_STATUSES.includes(task.status)) cancel(task.id);
    }
  }

  function clearCompleted() {
    tasks.value = tasks.value.filter(task => task.status !== 'success');
  }

  const hasActive = computed(() => tasks.value.some(task => ACTIVE_STATUSES.includes(task.status)));

  return { tasks, hasActive, enqueue, cancel, retry, cancelAll, clearCompleted };
}
