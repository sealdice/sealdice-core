import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseSettingSchema, getSdApiV2BaseSettingValue } from '@/api';

import { normalizeBaseSettingSchema } from './viewModel';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export function useBaseSettingSchemaQuery() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: ['base-setting-schema'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2BaseSettingSchema({ throwOnError: true });
      return normalizeBaseSettingSchema(data.item);
    },
  });
}

export function useBaseSettingValueQuery() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  return useQuery({
    queryKey: ['base-setting-value'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2BaseSettingValue({ throwOnError: true });
      return data.item;
    },
  });
}
