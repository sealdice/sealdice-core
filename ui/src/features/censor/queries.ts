import { computed, type MaybeRefOrGetter, toValue } from 'vue';
import { useQuery, type QueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2CensorConfig,
  getSdApiV2CensorFiles,
  getSdApiV2CensorLogsPage,
  getSdApiV2CensorStatus,
  getSdApiV2CensorWords,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import type { CensorLogQueryModel } from './viewModel';

export function useCensorStatusQuery() {
  return useQuery({
    queryKey: ['censor-status'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2CensorStatus({ throwOnError: true });
      return data.item;
    },
  });
}

export function useCensorConfigQuery(enabled: MaybeRefOrGetter<boolean>) {
  return useQuery({
    queryKey: ['censor-config'],
    enabled: computed(() => hasAccessToken.value && toValue(enabled)),
    queryFn: async () => {
      const { data } = await getSdApiV2CensorConfig({ throwOnError: true });
      return data.item;
    },
  });
}

export function useCensorFilesQuery(enabled: MaybeRefOrGetter<boolean>) {
  return useQuery({
    queryKey: ['censor-files'],
    enabled: computed(() => hasAccessToken.value && toValue(enabled)),
    queryFn: async () => {
      const { data } = await getSdApiV2CensorFiles({ throwOnError: true });
      return data.item.data ?? [];
    },
  });
}

export function useCensorWordsQuery(enabled: MaybeRefOrGetter<boolean>) {
  return useQuery({
    queryKey: ['censor-words'],
    enabled: computed(() => hasAccessToken.value && toValue(enabled)),
    queryFn: async () => {
      const { data } = await getSdApiV2CensorWords({ throwOnError: true });
      return data.item.data ?? [];
    },
  });
}

export function useCensorLogsQuery(
  query: MaybeRefOrGetter<CensorLogQueryModel>,
  enabled: MaybeRefOrGetter<boolean>,
) {
  const queryState = computed(() => ({ ...toValue(query) }));
  return useQuery({
    queryKey: computed(() => ['censor-logs', queryState.value]),
    enabled: computed(() => hasAccessToken.value && toValue(enabled)),
    queryFn: async () => {
      const { data } = await getSdApiV2CensorLogsPage({
        query: queryState.value,
        throwOnError: true,
      });
      return data.item;
    },
  });
}

export async function invalidateCensorQueries(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ['censor-status'] }),
    queryClient.invalidateQueries({ queryKey: ['censor-config'] }),
    queryClient.invalidateQueries({ queryKey: ['censor-files'] }),
    queryClient.invalidateQueries({ queryKey: ['censor-words'] }),
    queryClient.invalidateQueries({ queryKey: ['censor-logs'] }),
  ]);
}
