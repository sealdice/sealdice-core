import { computed, type MaybeRefOrGetter, toValue } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import {
  getSdApiV2ConfigAdvanced,
  getSdApiV2ConfigAdvancedQueryKey,
  getSdApiV2CustomTextOptions,
} from '@/api';

import { normalizeAdvancedConfig } from '@/features/config/advancedSettings';
import { appNavigation } from './navigation';
import { buildNavigationTree, flattenNavigationItems } from './navigationModel';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export function useAppNavigation(advancedConfigCounter: MaybeRefOrGetter<number>) {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  const customTextQuery = useQuery({
    ...getSdApiV2CustomTextOptions(),
    enabled: hasAccessToken,
  });
  const advancedConfigQuery = useQuery({
    queryKey: getSdApiV2ConfigAdvancedQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2ConfigAdvanced({
        throwOnError: true,
      });
      return normalizeAdvancedConfig(data.item);
    },
  });

  const customTextCategories = computed(() =>
    Object.keys(customTextQuery.data.value?.item.texts ?? {})
  );
  const advancedConfigEnabled = computed(() => {
    return toValue(advancedConfigCounter) >= 8 || advancedConfigQuery.data.value?.show === true;
  });

  const navigationTree = computed(() =>
    buildNavigationTree(appNavigation, {
      advancedConfigEnabled: advancedConfigEnabled.value,
      customTextCategories: customTextCategories.value,
    })
  );

  const searchItems = computed(() => flattenNavigationItems(navigationTree.value));

  return {
    advancedConfigQuery,
    customTextQuery,
    navigationTree,
    searchItems,
  };
}
