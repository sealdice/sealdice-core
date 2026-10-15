import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseSettingValue } from '@/api';

import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

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
