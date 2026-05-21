import { getApiBaseUrl } from '@/api/config';
import {
  postSdApiV2HelpdocUploadComplete,
  postSdApiV2HelpdocUploadInit,
} from '@/api';
import { useResumableUpload, type ResumableUploadTask } from '@/features/upload/resumableUpload';

export const helpdocChunkSize = 4 * 1024 * 1024;

export function useHelpdocUpload(options: {
  group: () => string;
  onSuccess: (task: ResumableUploadTask) => Promise<void> | void;
  onError: (task: ResumableUploadTask, error: unknown) => Promise<void> | void;
}) {
  return useResumableUpload('sd-helpdoc-upload-state', {
    chunkSize: helpdocChunkSize,
    async init(task) {
      const { data } = await postSdApiV2HelpdocUploadInit({
        body: {
          group: options.group(),
          filename: task.filename,
          fileSize: task.fileSize,
          fileHash: task.fileHash,
          chunkSize: helpdocChunkSize,
        },
        throwOnError: true,
      });
      return {
        sessionId: data.item.sessionId,
        chunkSize: data.item.chunkSize,
        uploadedChunks: data.item.uploadedChunks ?? [],
        uploadedBytes: data.item.uploadedBytes,
        expectedChunks: data.item.expectedChunks,
      };
    },
    async complete(task) {
      const { data } = await postSdApiV2HelpdocUploadComplete({
        body: {
          sessionId: task.sessionId,
        },
        throwOnError: true,
      });
      return data.item.success;
    },
    buildChunkUrl(task, index) {
      return `${getApiBaseUrl()}/sd-api/v2/helpdoc/upload/${encodeURIComponent(task.sessionId)}/${index}`;
    },
    onTaskSuccess: options.onSuccess,
    onTaskError: options.onError,
  });
}
