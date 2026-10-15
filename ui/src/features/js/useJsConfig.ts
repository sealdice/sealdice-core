import { computed } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsConfigsOptions,
  getSdApiV2JsConfigsQueryKey,
  getSdApiV2JsDeadConfigsOptions,
  getSdApiV2JsDeadConfigsQueryKey,
  postSdApiV2JsConfigs,
  postSdApiV2JsConfigsReset,
  postSdApiV2JsDeadConfigsDelete,
} from '@/api';

import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';

export function useJsConfig() {
  const { hasAccessToken } = storeToRefs(useAuthStore());
  const queryClient = useQueryClient();

  const configsQuery = useQuery({
    ...getSdApiV2JsConfigsOptions(),
    enabled: hasAccessToken,
    select: data => data.item,
  });

  const deadConfigsQuery = useQuery({
    ...getSdApiV2JsDeadConfigsOptions(),
    enabled: hasAccessToken,
    select: data => data.item.configs ?? [],
  });

  const configItems = computed(() =>
    Object.entries(configsQuery.data.value ?? {}).map(([name, cfg]) => ({
      pluginName: name,
      items: cfg?.configs ?? [],
    }))
  );

  const invalidateConfigs = () =>
    queryClient.invalidateQueries({ queryKey: getSdApiV2JsConfigsQueryKey() });
  const invalidateDeadConfigs = () =>
    queryClient.invalidateQueries({ queryKey: getSdApiV2JsDeadConfigsQueryKey() });

  const resetMutation = useMutation({
    mutationFn: async (payload: { name: string; keys: string[] }) => {
      await postSdApiV2JsConfigsReset({
        body: payload,
        throwOnError: true,
      });
    },
    onSuccess: async () => {
      await invalidateConfigs();
    },
  });

  const deleteDeadMutation = useMutation({
    mutationFn: async (names: string[]) => {
      await postSdApiV2JsDeadConfigsDelete({
        body: { names },
        throwOnError: true,
      });
    },
    onSuccess: async () => {
      await invalidateDeadConfigs();
      await invalidateConfigs();
    },
  });

  async function savePluginConfigs(configs: Record<string, Record<string, unknown>>) {
    for (const [name, config] of Object.entries(configs)) {
      if (!Object.keys(config).length) continue;
      await postSdApiV2JsConfigs({
        body: {
          name,
          config,
        },
        throwOnError: true,
      });
    }
    await invalidateConfigs();
  }

  return {
    configsQuery,
    deadConfigsQuery,
    configItems,
    resetMutation,
    deleteDeadMutation,
    savePluginConfigs,
  };
}
