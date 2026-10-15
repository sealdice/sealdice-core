import { computed, ref, shallowRef, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2ExtensionConfig,
  getSdApiV2ExtensionConfigSchema,
  putSdApiV2ExtensionConfig,
  type Instance,
} from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { requirePackageActionSuccess } from './operations';

const detailKey = (id: string) => ['extension-package-detail', id] as const;

export function usePackageDetail(options: {
  onSaved: () => unknown;
  onError: (error: unknown, fallback: string) => void;
}) {
  const auth = useAuthStore();
  const client = useQueryClient();
  const visible = ref(false);
  const pkg = shallowRef<Instance | null>(null);
  const id = computed(() => (visible.value ? (pkg.value?.manifest.package.id ?? '') : ''));
  // 按包隔离缓存，切换后旧请求的成功或失败都不能覆盖当前抽屉。
  const detail = useQuery({
    queryKey: computed(() => detailKey(id.value)),
    enabled: computed(() => auth.hasAccessToken && id.value !== ''),
    queryFn: async ({ signal, queryKey }) => {
      const query = { id: queryKey[1] };
      const [config, schema] = await Promise.all([
        getSdApiV2ExtensionConfig({ query, signal, throwOnError: true }),
        getSdApiV2ExtensionConfigSchema({ query, signal, throwOnError: true }),
      ]);
      return { config: config.data.item, schema: schema.data.item };
    },
  });
  const save = useMutation({
    mutationFn: async ({ id, config }: { id: string; config: Record<string, unknown> }) => {
      const { data } = await putSdApiV2ExtensionConfig({
        query: { id },
        body: config,
        throwOnError: true,
      });
      return requirePackageActionSuccess(data.item, '保存配置失败');
    },
    onSuccess: async (_result, submitted) => {
      // 只刷新真正保存的包；不把提交时的明文配置塞进其他包的状态。
      await client.invalidateQueries({ queryKey: detailKey(submitted.id), exact: true });
      await options.onSaved();
    },
    onError: error => options.onError(error, '保存配置失败'),
  });
  watch(detail.error, error => {
    if (!error) return;
    options.onError(error, '读取扩展包详情失败');
    visible.value = false;
  });
  watch(
    () => auth.hasAccessToken,
    canAccess => {
      if (!canAccess) {
        visible.value = false;
        pkg.value = null;
      }
    }
  );

  return {
    visible,
    pkg,
    config: computed(() => detail.data.value?.config ?? null),
    schema: computed(() => detail.data.value?.schema ?? null),
    loading: detail.isFetching,
    saving: save.isPending,
    open(next: Instance) {
      pkg.value = next;
      visible.value = true;
    },
    saveConfig(config: Record<string, unknown>) {
      if (id.value && !save.isPending.value) save.mutate({ id: id.value, config });
    },
  };
}
