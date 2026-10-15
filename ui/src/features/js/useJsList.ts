import { computed, type ComputedRef } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsListOptions,
  getSdApiV2JsListQueryKey,
  type GetSdApiV2JsListData,
  type GetSdApiV2JsListResponse,
  postSdApiV2JsCheckUpdate,
  postSdApiV2JsDelete,
  postSdApiV2JsDisable,
  postSdApiV2JsEnable,
  postSdApiV2JsUpdate,
} from '@/api';

import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export type JsUpdateDiff = {
  old: string;
  new: string;
  filename: string;
  tempFileName: string;
};

export function useJsList(options: {
  listParams: ComputedRef<NonNullable<GetSdApiV2JsListData['query']>>;
}) {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  const queryClient = useQueryClient();

  const listQueryResult = useQuery(
    computed(() => ({
      ...getSdApiV2JsListOptions({ query: options.listParams.value }),
      enabled: hasAccessToken.value,
      select: (data: GetSdApiV2JsListResponse) => data.item,
    }))
  );

  const invalidateList = () =>
    queryClient.invalidateQueries({ queryKey: getSdApiV2JsListQueryKey() });

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
