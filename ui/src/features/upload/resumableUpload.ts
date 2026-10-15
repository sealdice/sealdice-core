import { ref } from 'vue';
import * as tus from 'tus-js-client';
import { getApiBaseUrl } from '@/api/config';
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
  status: UploadTaskStatus;
  progress: number;
  uploadedBytes: number;
  errorText: string;
  // 内部取消标记，不参与渲染。
  canceled?: boolean;
};

export type ResumableUploadAdapter = {
  // 业务模块标识，作为 tus Upload-Metadata 的 module 字段。
  module: string;
  chunkSize?: number;
  // 额外的 tus 元数据（如帮助文档的 group）。
  metadata?: (task: ResumableUploadTask) => Record<string, string>;
  onTaskSuccess: (task: ResumableUploadTask) => Promise<void> | void;
  onTaskError?: (task: ResumableUploadTask, error: unknown) => Promise<void> | void;
};

// 是否处于可取消的进行中状态（排队/哈希/续传/上传/收尾）。
export function isResumableUploadActive(task: ResumableUploadTask): boolean {
  return task.status !== 'error' && task.status !== 'success';
}

type TusUploadOptions = ConstructorParameters<typeof tus.Upload>[1];
type TusPreviousUpload = {
  size: number | null;
  metadata: Record<string, string>;
  creationTime: string;
  urlStorageKey: string;
  uploadUrl: string | null;
  parallelUploadUrls: string[] | null;
};

const DEFAULT_CHUNK_SIZE = 4 * 1024 * 1024;
const UPLOAD_PATH = '/sd-api/v2/upload/';

// 通用分片上传控制器，底层使用 tus 协议（tus-js-client）。
// 断点续传由 tus-js-client + localStorage urlStorage 负责：刷新页面后只要用户
// 重新选择同一文件，就会按 fingerprint 找到之前的 upload URL 并从中断处续传。
export function useResumableUpload(storageKey: string, adapter: ResumableUploadAdapter) {
  const tasks = ref<ResumableUploadTask[]>([]);
  const busy = ref(false);
  const urlStorage = createLocalStorageUrlStorage(storageKey);
  // 记录进行中的 tus 上传，供取消使用。
  const activeUploads = new Map<string, { upload: tus.Upload; settle: () => void }>();

  function taskID(filename: string, fileSize: number, fileHash: string): string {
    return `${filename}:${fileSize}:${fileHash}`;
  }

  async function restore() {
    const previous = await urlStorage.findAllUploads();
    tasks.value = previous
      .filter(upload => upload.uploadUrl)
      .map(upload => {
        const filename = upload.metadata?.filename ?? '未命名文件';
        const fileSize = upload.size ?? 0;
        const fileHash = upload.metadata?.sha256 ?? '';
        return {
          id: taskID(filename, fileSize, fileHash),
          file: null,
          filename,
          fileSize,
          fileHash,
          status: 'error' as UploadTaskStatus,
          progress: 0,
          uploadedBytes: 0,
          errorText: '等待重新选择原文件以继续上传',
        };
      });
  }

  function removeTask(taskIDValue: string) {
    tasks.value = tasks.value.filter(task => task.id !== taskIDValue);
  }

  async function enqueueFiles(files: File[]) {
    for (const file of files) {
      // 先算内容 hash，再决定是否复用已有任务，避免同名不同内容误续传。
      const fileHash = await hashFile(file);
      const id = taskID(file.name, file.size, fileHash);
      const existing = tasks.value.find(task => task.id === id && task.status !== 'success');
      if (existing) {
        existing.file = file;
        existing.status = existing.uploadedBytes > 0 ? 'resuming' : 'queued';
        existing.errorText = '';
        continue;
      }
      tasks.value.unshift({
        id,
        file,
        filename: file.name,
        fileSize: file.size,
        fileHash,
        status: 'queued',
        progress: 0,
        uploadedBytes: 0,
        errorText: '',
      });
    }
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
    task.canceled = false;
    await start();
  }

  async function uploadTask(task: ResumableUploadTask) {
    try {
      if (!task.file) {
        task.status = 'error';
        task.errorText = '等待重新选择原文件以继续上传';
        return;
      }

      task.status = 'uploading';
      task.errorText = '';
      const metadata: Record<string, string> = {
        filename: task.filename,
        module: adapter.module,
        sha256: task.fileHash,
        ...(adapter.metadata ? adapter.metadata(task) : {}),
      };

      await new Promise<void>((resolve, reject) => {
        const upload = new tus.Upload(task.file as File, {
          endpoint: `${getApiBaseUrl()}${UPLOAD_PATH}`,
          chunkSize: adapter.chunkSize ?? DEFAULT_CHUNK_SIZE,
          retryDelays: [0, 1000, 3000, 5000, 10000],
          metadata,
          headers: authHeaders(),
          fingerprint: () => Promise.resolve(task.id),
          storeFingerprintForResuming: true,
          removeFingerprintOnSuccess: true,
          urlStorage: urlStorage as unknown as NonNullable<TusUploadOptions['urlStorage']>,
          onProgress: (bytesSent, bytesTotal) => {
            task.uploadedBytes = bytesSent;
            task.progress = bytesTotal
              ? Math.min(100, Math.floor((bytesSent / bytesTotal) * 100))
              : 0;
          },
          onSuccess: () => resolve(),
          onError: error => reject(error),
        });
        activeUploads.set(task.id, { upload, settle: resolve });

        void upload
          .findPreviousUploads()
          .then(previous => {
            if (previous.length > 0) {
              upload.resumeFromPreviousUpload(previous[0]);
            }
            upload.start();
          })
          .catch(reject);
      });

      if (task.canceled) return;
      task.status = 'success';
      task.progress = 100;
      task.uploadedBytes = task.fileSize;
      task.errorText = '';
      await adapter.onTaskSuccess(task);
      removeTask(task.id);
    } catch (error) {
      if (task.canceled) return;
      task.status = 'error';
      task.errorText = getUploadErrorText(error);
      await adapter.onTaskError?.(task, error);
    } finally {
      activeUploads.delete(task.id);
    }
  }

  async function cancelTask(task: ResumableUploadTask) {
    task.canceled = true;
    const active = activeUploads.get(task.id);
    if (active) {
      // 终止服务端上传，并释放挂起的 Promise。
      try {
        await active.upload.abort(true);
      } catch {
        // 忽略取消过程中的网络错误
      }
      active.settle();
    }
    activeUploads.delete(task.id);
    removeTask(task.id);
  }

  return {
    tasks,
    busy,
    restore,
    enqueueFiles,
    retry,
    cancelTask,
    removeTask,
    start,
  };
}

function authHeaders(): Record<string, string> {
  const token = currentAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function createLocalStorageUrlStorage(namespace: string) {
  const prefix = `${namespace}::upload::`;
  const keyFor = (fingerprint: string) => `${prefix}${fingerprint}`;

  return {
    async findAllUploads(): Promise<TusPreviousUpload[]> {
      if (typeof localStorage === 'undefined') return [];
      const result: TusPreviousUpload[] = [];
      for (let i = 0; i < localStorage.length; i += 1) {
        const key = localStorage.key(i);
        if (!key || !key.startsWith(prefix)) continue;
        const value = readStoredUpload(key);
        if (value) result.push(value);
      }
      return result;
    },
    async findUploadsByFingerprint(fingerprint: string): Promise<TusPreviousUpload[]> {
      const value = readStoredUpload(keyFor(fingerprint));
      return value ? [value] : [];
    },
    async removeUpload(urlStorageKey: string): Promise<void> {
      try {
        localStorage.removeItem(urlStorageKey);
      } catch {
        // ignore storage errors
      }
    },
    async addUpload(fingerprint: string, upload: TusPreviousUpload): Promise<string> {
      const key = keyFor(fingerprint);
      try {
        localStorage.setItem(key, JSON.stringify({ ...upload, urlStorageKey: key }));
      } catch {
        // ignore storage errors
      }
      return key;
    },
  };
}

function readStoredUpload(key: string): TusPreviousUpload | null {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return null;
    return JSON.parse(raw) as TusPreviousUpload;
  } catch {
    return null;
  }
}

function getUploadErrorText(error: unknown): string {
  if (error instanceof Error) {
    return error.message || '上传失败';
  }
  return '上传失败';
}
