import { computed } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseOverviewOptions } from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { formatAppChannel } from './appChannel';

export function useBaseOverview() {
  const overviewQuery = useQuery({
    ...getSdApiV2BaseOverviewOptions(),
    enabled: hasAccessToken,
  });

  const overview = computed(() => overviewQuery.data.value?.item);
  const appName = computed(() => overview.value?.appName || 'SealDice');
  const isStable = computed(() => overview.value?.appChannel === 'stable');
  const channelText = computed(() => formatAppChannel(overview.value?.appChannel));
  const hasNewVersion = computed(() => {
    const version = overview.value?.version;
    if (!version) return false;
    return version.code < version.latestCode;
  });

  return {
    overviewQuery,
    overview,
    appName,
    isStable,
    channelText,
    hasNewVersion,
  };
}
