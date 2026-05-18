import { computed } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseOverviewOptions } from '@/api';
import { hasAccessToken } from '@/features/auth/state';

export function useBaseOverview() {
  const overviewQuery = useQuery({
    ...getSdApiV2BaseOverviewOptions(),
    enabled: hasAccessToken,
  });

  const overview = computed(() => overviewQuery.data.value?.item);
  const appName = computed(() => overview.value?.appName || 'SealDice-CE');
  const runtimeText = computed(() =>
    overview.value ? `${overview.value.runtime.OS} - ${overview.value.runtime.arch}` : '',
  );
  const isStable = computed(() => overview.value?.appChannel === 'stable');
  const hasNewVersion = computed(() => {
    const version = overview.value?.version;
    if (!version) return false;
    return version.code < version.latestCode;
  });

  return {
    overviewQuery,
    overview,
    appName,
    runtimeText,
    isStable,
    hasNewVersion,
  };
}
