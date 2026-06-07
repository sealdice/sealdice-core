import { ref } from 'vue';
import { ApiError } from '@/api';
import { currentAccessToken } from '@/features/auth/state';
import { hashFile } from './fileHash';

export type UploadTaskStatus =
  | 'queued'
  | 'hashing'
  | 'resuming'
  | 'uploading'
  | 'completing'
  | 'success'
  | 'error';

export type ResumableUploadTask = {
  id: string;
  file: File | null;
  filename: string;
  fileSize: number;
  fileHash: string;
  sessionId: string;
  status: UploadTaskStatus;
  progress: number;
  uploadedChunks: number[];
  uploadedBytes: number;
  expectedChunks: number;
  errorText: string;
};

export type PersistedUploadTask = {
  filename: string;
  fileSize: number;
  fileHash: string;
  sessionId: string;
  uploadedChunks: number[];
  uploadedBytes: number;
  expectedChunks: number;
};

export type UploadSession = {
  sessionId: string;
  chunkSize: number;
  uploadedChunks: number[];
  uploadedBytes: number;
  expectedChunks: number;
};

export type ResumableUploadAdapter = {
  chunkSize: number;
  init(task: ResumableUploadTask): Promise<UploadSession>;
  complete(task: ResumableUploadTask): Promise<boolean>;
  buildChunkUrl(task: ResumableUploadTask, index: number): string;
  onTaskSuccess(task: ResumableUploadTask): Promise<void> | void;
  onTaskError?(task: ResumableUploadTask, error: unknown): Promise<void> | void;
};

// 通用分片上传控制器。业务页只实现 adapter，把“初始化会话、上传分片、
// 完成合并、成功刷新”这几个后端差异注入进来。
// 控制器负责文件哈希、断点恢复、本地持久化、串行上传和错误展示文案。
export function useResumableUpload(storageKey: string, adapter: ResumableUploadAdapter) {
  const tasks = ref<ResumableUploadTask[]>([]);
  const busy = ref(false);

  function restore() {
    // 只能持久化上传进度，不能持久化 File 对象。刷新页面后需要用户重新选择同一文件，
    // 再根据 filename/size/hash 继续已有 session。
    try {
      const raw = localStorage.getItem(storageKey);
      if (!raw) return;
      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) {
        localStorage.removeItem(storageKey);
        return;
      }
      const items = parsed.filter(isPersistedUploadTask) as PersistedUploadTask[];
      tasks.value = items.map(item => ({
        id: `${item.filename}:${item.fileSize}:${item.fileHash}`,
        file: null,
        filename: item.filename,
        fileSize: item.fileSize,
        fileHash: item.fileHash,
        sessionId: item.sessionId,
        status: 'error',
        progress: calcProgress(item.uploadedBytes, item.fileSize),
        uploadedChunks: item.uploadedChunks,
        uploadedBytes: item.uploadedBytes,
        expectedChunks: item.expectedChunks,
        errorText: '等待重新选择原文件以继续上传',
      }));
    } catch {
      safeRemove(storageKey);
    }
  }

  function persist() {
    const items = tasks.value
      .filter(task => task.status !== 'success')
      .map<PersistedUploadTask>(task => ({
        filename: task.filename,
        fileSize: task.fileSize,
        fileHash: task.fileHash,
        sessionId: task.sessionId,
        uploadedChunks: task.uploadedChunks,
        uploadedBytes: task.uploadedBytes,
        expectedChunks: task.expectedChunks,
      }));
    try {
      localStorage.setItem(storageKey, JSON.stringify(items));
    } catch {
      // ignore storage errors
    }
  }

  function removeTask(taskID: string) {
    tasks.value = tasks.value.filter(task => task.id !== taskID);
    persist();
  }

  async function enqueueFiles(files: File[]) {
    for (const file of files) {
      // 先算内容 hash，再决定是否复用已有任务，避免同名不同内容误续传。
      const fileHash = await hashFile(file);
      const existing = tasks.value.find(
        task =>
          task.filename === file.name &&
          task.fileSize === file.size &&
          task.fileHash === fileHash &&
          task.status !== 'success',
      );
      if (existing) {
        existing.file = file;
        existing.status = existing.uploadedChunks.length > 0 ? 'resuming' : 'queued';
        existing.errorText = '';
        continue;
      }
      tasks.value.unshift({
        id: `${file.name}:${file.size}:${file.lastModified}`,
        file,
        filename: file.name,
        fileSize: file.size,
        fileHash,
        sessionId: '',
        status: 'queued',
        progress: 0,
        uploadedChunks: [],
        uploadedBytes: 0,
        expectedChunks: 0,
        errorText: '',
      });
    }
    persist();
    await start();
  }

  async function start() {
    if (busy.value) return;
    busy.value = true;
    try {
      for (const task of tasks.value) {
        if (task.status === 'success') continue;
        await uploadTask(task);
      }
    } finally {
      busy.value = false;
    }
  }

  async function retry(task: ResumableUploadTask) {
    if (task.status !== 'error') return;
    task.status = 'queued';
    task.errorText = '';
    persist();
    await start();
  }

  async function uploadTask(task: ResumableUploadTask) {
    try {
      if (!task.file) {
        task.status = 'error';
        task.errorText = '等待重新选择原文件以继续上传';
        persist();
        return;
      }

      task.status = 'resuming';
      task.errorText = '';
      const session = await adapter.init(task);
      task.sessionId = session.sessionId;
      task.expectedChunks = session.expectedChunks;
      task.uploadedChunks = [...(session.uploadedChunks ?? [])];
      task.uploadedBytes = session.uploadedBytes;
      task.progress = calcProgress(task.uploadedBytes, task.fileSize);
      persist();

      task.status = 'uploading';
      for (let index = 0; index < task.expectedChunks; index++) {
        // 后端返回 uploadedChunks 时跳过已存在分片，实现真正的断点续传。
        if (task.uploadedChunks.includes(index)) continue;
        const start = index * session.chunkSize;
        const end = Math.min(task.fileSize, start + session.chunkSize);
        const chunk = task.file.slice(start, end);
        const chunkBytes = chunk.size;
        await uploadChunk(task, index, chunk, session.chunkSize, adapter.buildChunkUrl(task, index));
        task.uploadedChunks.push(index);
        task.uploadedBytes += chunkBytes;
        task.progress = calcProgress(task.uploadedBytes, task.fileSize);
        persist();
      }

      task.status = 'completing';
      const success = await adapter.complete(task);
      if (!success) {
        throw new Error('上传完成失败');
      }

      task.status = 'success';
      task.progress = 100;
      task.errorText = '';
      persist();
      await adapter.onTaskSuccess(task);
      removeTask(task.id);
    } catch (error) {
      task.status = 'error';
      task.errorText = getUploadErrorText(error);
      persist();
      await adapter.onTaskError?.(task, error);
    }
  }

  return {
    tasks,
    busy,
    restore,
    enqueueFiles,
    retry,
    removeTask,
    persist,
    start,
  };
}

async function uploadChunk(
  task: ResumableUploadTask,
  index: number,
  chunk: Blob,
  chunkSize: number,
  url: string,
) {
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', url, true);
    xhr.responseType = 'json';
    xhr.setRequestHeader('Content-Type', 'application/octet-stream');
    const token = currentAccessToken();
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    }
    xhr.upload.onprogress = event => {
      if (!event.lengthComputable) return;
      const baseBytes = task.uploadedChunks.reduce((sum, chunkIndex) => {
        const start = chunkIndex * chunkSize;
        const end = Math.min(task.fileSize, start + chunkSize);
        return sum + (end - start);
      }, 0);
      task.progress = calcProgress(baseBytes + event.loaded, task.fileSize);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }
      reject(
        new ApiError({
          status: xhr.status,
          statusText: xhr.statusText,
          data: xhr.response ?? xhr.responseText,
        }),
      );
    };
    xhr.onerror = () => reject(new Error('网络错误'));
    xhr.send(chunk);
  });

  void index;
}

function calcProgress(uploadedBytes: number, totalBytes: number): number {
  if (!totalBytes) return 0;
  return Math.min(100, Math.floor((uploadedBytes / totalBytes) * 100));
}

function isPersistedUploadTask(value: unknown): value is PersistedUploadTask {
  if (!value || typeof value !== 'object') return false;
  const item = value as Record<string, unknown>;
  return (
    typeof item.filename === 'string' &&
    typeof item.fileSize === 'number' &&
    typeof item.fileHash === 'string' &&
    typeof item.sessionId === 'string' &&
    Array.isArray(item.uploadedChunks) &&
    typeof item.uploadedBytes === 'number' &&
    typeof item.expectedChunks === 'number'
  );
}

function getUploadErrorText(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message || error.statusText || `HTTP ${error.status}`;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return '上传失败';
}

function safeRemove(key: string) {
  try {
    localStorage.removeItem(key);
  } catch {
    // ignore storage errors
  }
}
