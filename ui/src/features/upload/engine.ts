import { getUppy } from './uppy';

// 队列与具体上传引擎之间的窄接口：真实实现是 Uppy，测试用假引擎注入。
export type UploadEngineHandlers = {
  onProgress(taskId: string, bytesSent: number, bytesTotal: number): void;
  onSuccess(taskId: string): void;
  onError(taskId: string, error: Error): void;
};

export interface UploadEngine {
  setHandlers(handlers: UploadEngineHandlers): void;
  add(file: File, meta: Record<string, string>): void;
  start(): void;
  cancel(taskId: string): void;
  retry(taskId: string): void;
}

// createUppyEngine 把单例 Uppy 适配成 UploadEngine；用 meta.taskId 把事件映射回任务。
export function createUppyEngine(): UploadEngine {
  const uppy = getUppy();
  let handlers: UploadEngineHandlers | null = null;

  const taskIdOf = (file: { meta?: Record<string, unknown> } | undefined): string =>
    typeof file?.meta?.taskId === 'string' ? (file.meta.taskId as string) : '';

  uppy.on('upload-progress', (file, progress) => {
    const id = taskIdOf(file);
    if (id) handlers?.onProgress(id, progress.bytesUploaded ?? 0, progress.bytesTotal ?? 0);
  });
  uppy.on('upload-success', file => {
    const id = taskIdOf(file);
    if (id) handlers?.onSuccess(id);
  });
  uppy.on('upload-error', (file, error) => {
    const id = taskIdOf(file);
    if (id) handlers?.onError(id, error instanceof Error ? error : new Error(String(error)));
  });

  return {
    setHandlers(next) {
      handlers = next;
    },
    add(file, meta) {
      uppy.addFile({ name: meta.filename ?? file.name, type: file.type, data: file, meta });
    },
    start() {
      void uppy.upload().catch(() => {
        // 没有待上传文件时 Uppy 会 reject，忽略即可。
      });
    },
    cancel(taskId) {
      const target = uppy.getFiles().find(file => file.meta?.taskId === taskId);
      if (target) uppy.removeFile(target.id);
    },
    retry(taskId) {
      const target = uppy.getFiles().find(file => file.meta?.taskId === taskId);
      if (target) uppy.retryUpload(target.id);
    },
  };
}
