import { computed, type MaybeRefOrGetter, toValue } from 'vue';
import { useQuery, type QueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2HelpdocConfig,
  getSdApiV2HelpdocItemsPage,
  getSdApiV2HelpdocTree,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import {
  buildHelpdocGroupOptions,
  buildHelpdocItemParams,
  convertHelpdocTree,
  getHelpdocItemGroupOptions,
} from './viewModel';

export type HelpdocItemQueryModel = {
  pageNum: number;
  pageSize: number;
  id: number | null;
  group: string | null;
  from: string;
  title: string;
};

export function createDefaultHelpdocItemQuery(): HelpdocItemQueryModel {
  return {
    pageNum: 1,
    pageSize: 20,
    id: null,
    group: null,
    from: '',
    title: '',
  };
}

export function useHelpdocQueries(itemQuery: MaybeRefOrGetter<HelpdocItemQueryModel>) {
  const treeQuery = useQuery({
    queryKey: ['helpdoc-tree'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2HelpdocTree({ throwOnError: true });
      return data.item.data ?? [];
    },
  });

  const configQuery = useQuery({
    queryKey: ['helpdoc-config'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2HelpdocConfig({ throwOnError: true });
      return data.item.aliases ?? {};
    },
  });

  const itemParams = computed(() => buildHelpdocItemParams(toValue(itemQuery)));

  const itemsQuery = useQuery({
    queryKey: computed(() => ['helpdoc-items', itemParams.value]),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2HelpdocItemsPage({
        query: itemParams.value,
        throwOnError: true,
      });
      return data.item;
    },
  });

  const rawTree = computed(() => treeQuery.data.value ?? []);
  const docTree = computed(() => rawTree.value.map(convertHelpdocTree));
  const docGroups = computed(() => buildHelpdocGroupOptions(rawTree.value));
  const itemGroupOptions = computed(() => getHelpdocItemGroupOptions(rawTree.value));
  const helpItems = computed(() => itemsQuery.data.value?.data ?? []);
  const itemTotal = computed(() => itemsQuery.data.value?.total ?? 0);

  return {
    treeQuery,
    configQuery,
    itemsQuery,
    rawTree,
    docTree,
    docGroups,
    itemGroupOptions,
    helpItems,
    itemTotal,
  };
}

export async function invalidateHelpdocQueries(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ['helpdoc-tree'] }),
    queryClient.invalidateQueries({ queryKey: ['helpdoc-items'] }),
    queryClient.invalidateQueries({ queryKey: ['helpdoc-config'] }),
  ]);
}
