import { useQuery } from '@tanstack/vue-query';
import {
  getSdApiV2BaseSettingSchema,
  getSdApiV2BaseSettingValue,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { normalizeBaseSettingSchema } from './viewModel';

export function useBaseSettingSchemaQuery() {
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
  return useQuery({
    queryKey: ['base-setting-value'],
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2BaseSettingValue({ throwOnError: true });
      return data.item;
    },
  });
}
