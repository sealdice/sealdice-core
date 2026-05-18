import { computed, type MaybeRefOrGetter, toValue, watchEffect } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2ConfigAdvancedOptions, getSdApiV2CustomTextOptions } from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { setAdvancedSettingsVisible } from '@/features/config/advancedSettings';
import { appNavigation } from './navigation';
import { buildNavigationTree, flattenNavigationItems } from './navigationModel';

export function useAppNavigation(advancedConfigCounter: MaybeRefOrGetter<number>) {
  const customTextQuery = useQuery({
    ...getSdApiV2CustomTextOptions(),
    enabled: hasAccessToken,
  });
  const advancedConfigQuery = useQuery({
    ...getSdApiV2ConfigAdvancedOptions(),
    enabled: hasAccessToken,
  });

  const customTextCategories = computed(() =>
    Object.keys(customTextQuery.data.value?.item.texts ?? {}),
  );
  const advancedConfigEnabled = computed(() => {
    return toValue(advancedConfigCounter) >= 8 || advancedConfigQuery.data.value?.item.show === true;
  });

  watchEffect(() => {
    setAdvancedSettingsVisible(advancedConfigQuery.data.value?.item.show === true);
  });

  const navigationTree = computed(() =>
    buildNavigationTree(appNavigation, {
      advancedConfigEnabled: advancedConfigEnabled.value,
      customTextCategories: customTextCategories.value,
    }),
  );

  const searchItems = computed(() => flattenNavigationItems(navigationTree.value));

  return {
    advancedConfigQuery,
    customTextQuery,
    navigationTree,
    searchItems,
  };
}
