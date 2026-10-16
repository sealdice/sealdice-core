import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import { useMutation, useQuery } from '@tanstack/vue-query';
import { useAuthStore } from '@/features/auth/store';
import { getErrorMessage } from '@/features/auth/error';
import {
  getSdApiV2ExtensionPackages,
  getSdApiV2ExtensionStoreBackends,
  getSdApiV2ExtensionStorePage,
  postSdApiV2ExtensionPackagesRefresh,
} from '@/api';

type StoreFilter = { name: string; category?: string; pageNum: number; pageSize: number };

// 目录查询、取消、认证和刷新共用同一入口；页面只提供筛选条件。
export function usePackageCatalog(filter: MaybeRefOrGetter<StoreFilter>) {
  const auth = useAuthStore();
  const enabled = computed(() => auth.hasAccessToken);
  const packagesQuery = useQuery({
    queryKey: ['extension-packages'],
    enabled,
    queryFn: async ({ signal }) =>
      (await getSdApiV2ExtensionPackages({ signal, throwOnError: true })).data.item.items ?? [],
  });
  const storeBackendsQuery = useQuery({
    queryKey: ['extension-store-backends'],
    enabled,
    queryFn: async ({ signal }) =>
      (await getSdApiV2ExtensionStoreBackends({ signal, throwOnError: true })).data.item.items ??
      [],
  });
  const storePageQuery = useQuery({
    queryKey: computed(() => ['extension-store-page', toValue(filter)]),
    enabled,
    queryFn: async ({ signal }) => {
      const query = { ...toValue(filter) };
      return (await getSdApiV2ExtensionStorePage({ query, signal, throwOnError: true })).data.item;
    },
  });
  const refreshMutation = useMutation({
    mutationFn: async () => {
      await postSdApiV2ExtensionPackagesRefresh({ throwOnError: true });
      await packagesQuery.refetch({ throwOnError: true });
    },
  });
  const refreshing = computed(
    () =>
      packagesQuery.isFetching.value ||
      storeBackendsQuery.isFetching.value ||
      storePageQuery.isFetching.value
  );
  const loadErrorText = computed(() => {
    for (const [query, fallback] of [
      [packagesQuery, '扩展包读取失败'],
      [storeBackendsQuery, '仓库后端读取失败'],
      [storePageQuery, '商店列表读取失败'],
    ] as const)
      if (query.isError.value) return getErrorMessage(query.error.value, fallback);
    return '';
  });
  async function refreshAll() {
    await Promise.all([
      packagesQuery.refetch(),
      storeBackendsQuery.refetch(),
      storePageQuery.refetch(),
    ]);
  }
  return {
    packagesQuery,
    storeBackendsQuery,
    storePageQuery,
    loadErrorText,
    refreshing,
    storeLoading: storePageQuery.isFetching,
    packagesLoading: refreshMutation.isPending,
    refreshAll,
    refreshPackages: refreshMutation.mutateAsync,
    fetchStorePage: () => storePageQuery.refetch(),
  };
}
