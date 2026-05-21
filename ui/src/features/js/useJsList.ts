import { computed, type ComputedRef } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsList,
  postSdApiV2JsCheckUpdate,
  postSdApiV2JsDelete,
  postSdApiV2JsDisable,
  postSdApiV2JsEnable,
  postSdApiV2JsUpdate,
  postSdApiV2JsUploadComplete,
  postSdApiV2JsUploadInit,
} from '@/api';
import { getApiBaseUrl } from '@/api/config';
import { hasAccessToken } from '@/features/auth/state';
import { useResumableUpload, type ResumableUploadTask } from '@/features/upload/resumableUpload';
import { jsListQueryKey, type JsListParams } from './queryKeys';

const jsChunkSize = 4 * 1024 * 1024;

export type JsUpdateDiff = {
  old: string;
  new: string;
  filename: string;
  tempFileName: string;
};

export function useJsList(options: {
  listParams: ComputedRef<JsListParams>;
  onUploadSuccess: (task: ResumableUploadTask) => Promise<void> | void;
  onUploadError: (task: ResumableUploadTask) => Promise<void> | void;
}) {
  const queryClient = useQueryClient();

  const listQueryResult = useQuery({
    queryKey: computed(() => jsListQueryKey(options.listParams.value)),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2JsList({
        query: options.listParams.value,
        throwOnError: true,
      });
      return data.item;
    },
  });

  const invalidateList = () => queryClient.invalidateQueries({ queryKey: ['js-list'] });

  const deleteMutation = useMutation({
    mutationFn: async (filename: string) => {
      await postSdApiV2JsDelete({
        body: { filename },
        throwOnError: true,
      });
    },
  });

  const enableMutation = useMutation({
    mutationFn: async (name: string) => {
      await postSdApiV2JsEnable({
        body: { name },
        throwOnError: true,
      });
    },
  });

  const disableMutation = useMutation({
    mutationFn: async (name: string) => {
      await postSdApiV2JsDisable({
        body: { name },
        throwOnError: true,
      });
    },
  });

  const uploader = useResumableUpload('sd-js-upload-state', {
    chunkSize: jsChunkSize,
    async init(task: ResumableUploadTask) {
      const { data } = await postSdApiV2JsUploadInit({
        body: {
          filename: task.filename,
          fileSize: task.fileSize,
          fileHash: task.fileHash,
          chunkSize: jsChunkSize,
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
    async complete(task: ResumableUploadTask) {
      const { data } = await postSdApiV2JsUploadComplete({
        body: {
          sessionId: task.sessionId,
        },
        throwOnError: true,
      });
      return data.item.success;
    },
    buildChunkUrl(task: ResumableUploadTask, index: number) {
      return `${getApiBaseUrl()}/sd-api/v2/js/upload/${encodeURIComponent(task.sessionId)}/${index}`;
    },
    onTaskSuccess: options.onUploadSuccess,
    onTaskError: options.onUploadError,
  });

  async function checkUpdate(filename: string): Promise<JsUpdateDiff | null> {
    const { data } = await postSdApiV2JsCheckUpdate({
      body: { filename },
      throwOnError: true,
    });
    if (!data.item.success) {
      throw new Error(data.item.err || '检查更新失败');
    }
    return {
      old: data.item.old || '',
      new: data.item.new || '',
      filename: data.item.filename || '',
      tempFileName: data.item.tempFileName || '',
    };
  }

  async function applyUpdate(diff: JsUpdateDiff) {
    await postSdApiV2JsUpdate({
      body: {
        filename: diff.filename,
        tempFileName: diff.tempFileName,
      },
      throwOnError: true,
    });
  }

  return {
    listQueryResult,
    invalidateList,
    deleteMutation,
    enableMutation,
    disableMutation,
    uploader,
    checkUpdate,
    applyUpdate,
  };
}
