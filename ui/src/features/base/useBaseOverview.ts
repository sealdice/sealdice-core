import { computed } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseOverviewOptions } from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { formatAppChannel, getAppChannelHint, getAppChannelTagType } from './appChannel';
import { formatRuntimeSummary } from './runtimeSummary';
import { formatDisplayVersion } from './versionDisplay';

export function useBaseOverview() {
  const overviewQuery = useQuery({
    ...getSdApiV2BaseOverviewOptions(),
    enabled: hasAccessToken,
  });

  const overview = computed(() => overviewQuery.data.value?.item);
  const appName = computed(() => overview.value?.appName || 'SealDice');
  const appChannel = computed(() => overview.value?.appChannel);
  const isStable = computed(() => appChannel.value === 'stable');
  const channelText = computed(() => formatAppChannel(appChannel.value));
  const channelTagType = computed(() => getAppChannelTagType(appChannel.value));
  const channelHint = computed(() => getAppChannelHint(appChannel.value));
  const displayVersion = computed(() =>
    formatDisplayVersion(overview.value?.version, appChannel.value)
  );
  const runtimeText = computed(() => formatRuntimeSummary(overview.value?.runtime));
  const hasNewVersion = computed(() => {
    const version = overview.value?.version;
    if (!version) return false;
    return version.code < version.latestCode;
  });

  return {
    overviewQuery,
    overview,
    appName,
    appChannel,
    isStable,
    channelText,
    channelTagType,
    channelHint,
    displayVersion,
    runtimeText,
    hasNewVersion,
  };
}
