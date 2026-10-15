import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import {
  getSdApiV2ImconnectionByIdConfig,
  getSdApiV2Imconnection,
  getSdApiV2ImconnectionProtocols,
  getSdApiV2ImconnectionSchemas,
  getSdApiV2ImconnectionSignInfo,
} from '@/api';

import {
  connectEndpointConfigQueryKey,
  connectProtocolsQueryKey,
  connectSchemasQueryKey,
  connectSignInfoQueryKey,
  connectionsQueryKey,
} from './queryKeys';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export function useConnectConnectionsQuery() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: connectionsQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2Imconnection({ throwOnError: true });
      return data;
    },
  });
}

export function useConnectProtocolsQuery() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: connectProtocolsQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2ImconnectionProtocols({ throwOnError: true });
      return data;
    },
  });
}

export function useConnectSchemasQuery() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: connectSchemasQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2ImconnectionSchemas({ throwOnError: true });
      return data;
    },
  });
}

export function useConnectSignInfoQuery(enabled: MaybeRefOrGetter<boolean>) {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: connectSignInfoQueryKey(),
    enabled: computed(() => hasAccessToken.value && toValue(enabled)),
    queryFn: async () => {
      const { data } = await getSdApiV2ImconnectionSignInfo({ throwOnError: true });
      return data;
    },
  });
}

export function useConnectEndpointConfigQuery(
  endpointId: MaybeRefOrGetter<string>,
  enabled: MaybeRefOrGetter<boolean>
) {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  const id = computed(() => toValue(endpointId));
  return useQuery({
    queryKey: computed(() => connectEndpointConfigQueryKey(id.value)),
    enabled: computed(() => hasAccessToken.value && toValue(enabled) && id.value !== ''),
    queryFn: async () => {
      const { data } = await getSdApiV2ImconnectionByIdConfig({
        path: { id: id.value },
        throwOnError: true,
      });
      return data.item;
    },
  });
}
