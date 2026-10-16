import { computed, type Ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsByNameDataInfoOptions,
  getSdApiV2JsByNameDataInfoQueryKey,
  getSdApiV2JsByNameDataListOptions,
  getSdApiV2JsByNameDataListQueryKey,
  getSdApiV2JsListOptions,
  postSdApiV2JsByNameDataMutation,
  postSdApiV2JsByNameDataDeleteMutation,
  postSdApiV2JsByNameDataShrinkMutation,
  type GetSdApiV2JsByNameDataListResponse,
  type GetSdApiV2JsByNameDataInfoResponse,
} from '@/api';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export function useJsData(options: {
  selectedPlugin: Ref<string>;
  dataPage: Ref<{ page: number; pageSize: number }>;
  dataKeyword: Ref<string>;
}) {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  const queryClient = useQueryClient();

  const pluginListQuery = useQuery({
    ...getSdApiV2JsListOptions({ query: { page: 1, pageSize: 200 } }),
    enabled: hasAccessToken,
    select: data => data.item.data ?? [],
  });

  const pluginOptions = computed(() =>
    (pluginListQuery.data.value ?? []).map(plugin => ({
      label: `${plugin.name} (v${plugin.version})`,
      value: plugin.name,
    }))
  );

  const dataListQuery = useQuery(
    computed(() => ({
      ...getSdApiV2JsByNameDataListOptions({
        path: { name: options.selectedPlugin.value },
        query: { ...options.dataPage.value, keyword: options.dataKeyword.value || undefined },
      }),
      enabled: hasAccessToken.value && !!options.selectedPlugin.value,
      select: (data: GetSdApiV2JsByNameDataListResponse) => data.item,
    }))
  );

  const dataInfoQuery = useQuery(
    computed(() => ({
      ...getSdApiV2JsByNameDataInfoOptions({ path: { name: options.selectedPlugin.value } }),
      enabled: hasAccessToken.value && !!options.selectedPlugin.value,
      select: (data: GetSdApiV2JsByNameDataInfoResponse) => data.item,
    }))
  );

  // 写入目标来自提交参数；完成回调不能重新读取可能已经切换的选中插件。
  const setMutation = useMutation({
    ...postSdApiV2JsByNameDataMutation(),
    onSuccess: (_data, { path }) => invalidateData(path.name),
  });
  const deleteMutation = useMutation({
    ...postSdApiV2JsByNameDataDeleteMutation(),
    onSuccess: (_data, { path }) => invalidateData(path.name),
  });
  const shrinkMutation = useMutation({
    ...postSdApiV2JsByNameDataShrinkMutation(),
    onSuccess: (_data, { path }) =>
      queryClient.invalidateQueries({ queryKey: getSdApiV2JsByNameDataInfoQueryKey({ path }) }),
  });

  function invalidateData(name = options.selectedPlugin.value) {
    const path = { name };
    return Promise.all([
      queryClient.invalidateQueries({ queryKey: getSdApiV2JsByNameDataListQueryKey({ path }) }),
      queryClient.invalidateQueries({ queryKey: getSdApiV2JsByNameDataInfoQueryKey({ path }) }),
    ]);
  }

  return {
    pluginListQuery,
    pluginOptions,
    dataListQuery,
    dataInfoQuery,
    setMutation,
    deleteMutation,
    shrinkMutation,
    invalidateData,
  };
}
