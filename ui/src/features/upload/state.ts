import { computed } from 'vue';
import { appPinia } from '@/pinia';
import { useUploadStore } from './store';
import type { EnqueueContext } from './queue';
import type { UploadKindID } from './kinds';

const store = useUploadStore(appPinia);

export const uploadTasks = computed(() => store.tasks);
export const hasActiveUploads = computed(() => store.hasActive);

export function enqueueUploads(
  files: File[],
  kind: UploadKindID,
  ctx?: EnqueueContext
): Promise<void> {
  return store.enqueue(files, kind, ctx);
}

export function cancelUpload(id: string): void {
  store.cancel(id);
}

export function retryUpload(id: string): void {
  store.retry(id);
}

export function cancelAllUploads(): void {
  store.cancelAll();
}

export function clearCompletedUploads(): void {
  store.clearCompleted();
}

export type { UploadTask, UploadTaskStatus, EnqueueContext } from './queue';
