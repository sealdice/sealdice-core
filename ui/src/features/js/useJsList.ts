import { computed, type ComputedRef } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsList,
  postSdApiV2JsCheckUpdate,
  postSdApiV2JsDelete,
  postSdApiV2JsDisable,
  postSdApiV2JsEnable,
  postSdApiV2JsUpdate,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { jsListQueryKey, type JsListParams } from './queryKeys';

export type JsUpdateDiff = {
  old: string;
  new: string;
  filename: string;
  tempFileName: string;
};

export function useJsList(options: { listParams: ComputedRef<JsListParams> }) {
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
    checkUpdate,
    applyUpdate,
  };
}
