import Uppy from '@uppy/core';
import Tus from '@uppy/tus';
import { getApiBaseUrl } from '@/api/config';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';

// The Pinia upload store owns this instance; tests can create and dispose their own.
export function createUppy(): Uppy {
  return new Uppy({
    autoProceed: false,
    allowMultipleUploadBatches: true,
    onBeforeFileAdded(file, files) {
      // Stable across reloads for tus resume, but distinct for different destinations.
      const id = JSON.stringify([file.id, file.meta.module, file.meta.group, file.meta.sha256]);
      if (files[id] && !files[id].progress.uploadComplete)
        throw new Error('相同目标的文件已在上传队列中，请等待完成或取消后再添加');
      return { ...file, id };
    },
  }).use(Tus, {
    endpoint: `${getApiBaseUrl()}/sd-api/v2/upload/`,
    retryDelays: [0, 1000, 3000, 5000, 10000],
    withCredentials: false,
    storeFingerprintForResuming: true,
    removeFingerprintOnSuccess: true,
    headers: (): Record<string, string> => {
      const token = useAuthStore(appPinia).token;
      return token ? { Authorization: `Bearer ${token}` } : {};
    },
    onShouldRetry(error, _attempt, _options, next) {
      const status = error.originalResponse?.getStatus();
      return status !== undefined && status >= 400 && status < 500 ? false : next(error);
    },
  });
}
