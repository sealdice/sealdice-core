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
    instance = new Uppy({ autoProceed: true, allowMultipleUploadBatches: true });
    instance.use(Tus, {
      endpoint: uploadEndpoint(),
      retryDelays: [0, 1000, 3000, 5000, 10000],
      withCredentials: false,
      // 透传给 tus-js-client：刷新后重选同一文件可续传（见规格 §10 任务 6 结论）。
      storeFingerprintForResuming: true,
      headers: () => authHeaders(),
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
