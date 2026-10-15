import Uppy from '@uppy/core';
import Tus from '@uppy/tus';
import { getApiBaseUrl } from '@/api/config';
import { currentAccessToken } from '@/features/auth/state';

// 全应用唯一的 tus 上传端点（后端已统一）。
const UPLOAD_PATH = '/sd-api/v2/upload/';

let instance: Uppy | null = null;

// getUppy 返回惰性创建的单例。全应用共享同一个队列/引擎。
export function getUppy(): Uppy {
  if (!instance) {
    instance = new Uppy({
      autoProceed: false,
      allowMultipleUploadBatches: true,
      // 允许重复文件：Uppy 默认用派生 id 去重，成功后文件仍留在实例里，
      // 会导致同一文件二次入队抛 RestrictionError（队列卡死）。上传去重由业务层负责。
      onBeforeFileAdded: () => true,
    });
    instance.use(Tus, {
      endpoint: uploadEndpoint(),
      retryDelays: [0, 1000, 3000, 5000, 10000],
      withCredentials: false,
      // 透传给 tus-js-client：刷新后重选同一文件可续传（见规格 §10 任务 6 结论）。
      storeFingerprintForResuming: true,
      headers: () => authHeaders(),
      // 4xx（如 409 同名冲突）是确定性失败，不重试。
      onShouldRetry: (
        error: { originalResponse?: { getStatus?: () => number } },
        _attempt: number,
        _options: unknown,
        next: (err: unknown) => boolean
      ) => {
        const status = error?.originalResponse?.getStatus?.();
        if (typeof status === 'number' && status >= 400 && status < 500) return false;
        return next(error);
      },
    } as never);
  }
  return instance;
}

export function uploadEndpoint(): string {
  return `${getApiBaseUrl()}${UPLOAD_PATH}`;
}

export function authHeaders(): Record<string, string> {
  const token = currentAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
