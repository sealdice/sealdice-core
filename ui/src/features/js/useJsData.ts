import { computed, type Ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsByNameDataInfo,
  getSdApiV2JsByNameDataList,
  getSdApiV2JsList,
  postSdApiV2JsByNameData,
  postSdApiV2JsByNameDataDelete,
  postSdApiV2JsByNameDataShrink,
  type JsInfo,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import {
  jsDataInfoQueryKey,
  jsDataListQueryKey,
  jsListForDataQueryKey,
  type JsDataPage,
} from './queryKeys';

export function useJsData(options: {
  selectedPlugin: Ref<string>;
  dataPage: Ref<JsDataPage>;
  dataKeyword: Ref<string>;
}) {
  const queryClient = useQueryClient();

  const pluginListQuery = useQuery({
    queryKey: jsListForDataQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2JsList({
        query: { page: 1, pageSize: 200 },
        throwOnError: true,
      });
      return data.item.data ?? [];
    },
  });

  const pluginOptions = computed(() =>
    (pluginListQuery.data.value ?? []).map((p: JsInfo) => ({
      label: `${p.name} (v${p.version})`,
      value: p.name,
    })),
  );

  const dataListQuery = useQuery({
    queryKey: computed(() =>
      jsDataListQueryKey(options.selectedPlugin.value, options.dataPage.value, options.dataKeyword.value),
    ),
    enabled: computed(() => hasAccessToken.value && !!options.selectedPlugin.value),
    queryFn: async () => {
      const { data } = await getSdApiV2JsByNameDataList({
        path: { name: options.selectedPlugin.value },
        query: {
          page: options.dataPage.value.page,
          pageSize: options.dataPage.value.pageSize,
          keyword: options.dataKeyword.value || undefined,
        },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const dataInfoQuery = useQuery({
    queryKey: computed(() => jsDataInfoQueryKey(options.selectedPlugin.value)),
    enabled: computed(() => hasAccessToken.value && !!options.selectedPlugin.value),
    queryFn: async () => {
      const { data } = await getSdApiV2JsByNameDataInfo({
        path: { name: options.selectedPlugin.value },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const setMutation = useMutation({
    mutationFn: async (payload: { key: string; value: string }) => {
      await postSdApiV2JsByNameData({
        path: { name: options.selectedPlugin.value },
        body: payload,
        throwOnError: true,
      });
    },
    onSuccess: () => {
      invalidateData();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (keys: string[]) => {
      await postSdApiV2JsByNameDataDelete({
        path: { name: options.selectedPlugin.value },
        body: { keys },
        throwOnError: true,
      });
    },
    onSuccess: () => {
      invalidateData();
    },
  });

  const shrinkMutation = useMutation({
    mutationFn: async () => {
      await postSdApiV2JsByNameDataShrink({
        path: { name: options.selectedPlugin.value },
        throwOnError: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jsDataInfoQueryKey(options.selectedPlugin.value) });
    },
  });

  function invalidateData() {
    queryClient.invalidateQueries({ queryKey: ['js-data-list'] });
    queryClient.invalidateQueries({ queryKey: ['js-data-info'] });
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
