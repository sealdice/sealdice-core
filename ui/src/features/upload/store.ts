import { defineStore } from 'pinia';
import { onScopeDispose } from 'vue';
import { createUppy } from './uppy';
import { createUppyEngine } from './engine';
import { createUploadQueue } from './queue';
import { hashFile } from './fileHash';

export const useUploadStore = defineStore('upload', () => {
  const uppy = createUppy();
  const queue = createUploadQueue({ engine: createUppyEngine(uppy), hashFile });
  onScopeDispose(() => {
    for (const task of queue.tasks.value) queue.cancel(task.id);
    uppy.destroy();
  });
  return queue;
});
