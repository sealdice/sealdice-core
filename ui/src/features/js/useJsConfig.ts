import { computed } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2JsConfigs,
  getSdApiV2JsDeadConfigs,
  postSdApiV2JsConfigs,
  postSdApiV2JsConfigsReset,
  postSdApiV2JsDeadConfigsDelete,
  type ApiPluginConfig,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { jsConfigsQueryKey, jsDeadConfigsQueryKey } from './queryKeys';

export function useJsConfig() {
  const queryClient = useQueryClient();

  const configsQuery = useQuery({
    queryKey: jsConfigsQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2JsConfigs({ throwOnError: true });
      return data.item;
    },
  });

  const deadConfigsQuery = useQuery({
    queryKey: jsDeadConfigsQueryKey(),
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2JsDeadConfigs({ throwOnError: true });
      return data.item.configs ?? [];
    },
  });

  const configEntries = computed<[string, ApiPluginConfig][]>(() => {
    const map = (configsQuery.data.value ?? {}) as Record<string, ApiPluginConfig>;
    return Object.entries(map);
  });

  const configItems = computed(() => {
    return configEntries.value.map(([name, cfg]) => ({
      pluginName: name,
      items: cfg.configs ?? [],
    }));
  });

  const invalidateConfigs = () => queryClient.invalidateQueries({ queryKey: jsConfigsQueryKey() });
  const invalidateDeadConfigs = () => queryClient.invalidateQueries({ queryKey: jsDeadConfigsQueryKey() });

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
