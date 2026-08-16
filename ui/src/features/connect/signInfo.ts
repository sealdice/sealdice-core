import { computed, watch, type Ref } from 'vue';
import type { ProtocolDefinition } from '@/api';
import type { SelectOption } from 'naive-ui';
import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
import { getConnectProtocolModule } from './protocols';
import { useConnectSignInfoQuery } from './queries';
import { buildSignInfoState } from './signInfoState';

export function useConnectSignInfo(
  protocol: Readonly<Ref<ProtocolDefinition | null>>,
  formModel: Ref<DynamicFormModel>
) {
  const enabled = computed(
    () => getConnectProtocolModule(protocol.value?.key ?? '').formKind === 'sign-info'
  );
  const query = useConnectSignInfoQuery(enabled);

  watch(query.data, data => {
    if (!enabled.value) return;
    const items = data?.item.items ?? [];
    const selectedVersion =
      items.find(item => item.selected && !item.ignored) ?? items.find(item => !item.ignored);
    const selectedServer =
      selectedVersion?.servers?.find(item => item.selected && !item.ignored) ??
      selectedVersion?.servers?.find(item => !item.ignored);
    formModel.value = {
      ...formModel.value,
      signServerVersion: selectedVersion?.version ?? '',
      signServerName: selectedServer?.name ?? '',
    };
  });

  const state = computed(() =>
    buildSignInfoState({
      enabled: enabled.value,
      isLoading: query.isLoading.value,
      isFetching: query.isFetching.value,
      isError: query.isError.value,
      hasData: (query.data.value?.item.items?.length ?? 0) > 0,
      signServerVersion: String(formModel.value.signServerVersion ?? ''),
    })
  );

  const versions = computed<SelectOption[]>(() =>
    (query.data.value?.item.items ?? [])
      .filter(item => !item.ignored)
      .map(item => ({
        label: item.selected ? `${item.version} 最新` : item.version,
        value: item.version,
      }))
      .concat({ label: '自定义', value: '自定义' })
  );

  const servers = computed<SelectOption[]>(() => {
    const version = formModel.value.signServerVersion;
    const info = (query.data.value?.item.items ?? []).find(item => item.version === version);
    return (info?.servers ?? [])
      .filter(item => !item.ignored)
      .map(item => ({
        label: item.latency > 0 ? `${item.name} (${item.latency}ms)` : item.name,
        value: item.name,
      }));
  });

  watch(
    () => formModel.value.signServerVersion,
    version => {
      if (!enabled.value || version === '自定义') return;
      const info = (query.data.value?.item.items ?? []).find(item => item.version === version);
      const server =
        info?.servers?.find(item => item.selected && !item.ignored) ??
        info?.servers?.find(item => !item.ignored);
      if (server && formModel.value.signServerName !== server.name) {
        formModel.value = {
          ...formModel.value,
          signServerName: server.name,
        };
      }
    }
  );

  const versionOptions = computed<SelectOption[]>(() =>
    state.value.mode === 'manual-fallback' ? [{ label: '自定义', value: '自定义' }] : versions.value
  );

  const errorMessage = computed(() =>
    state.value.mode === 'manual-fallback' ? state.value.message : ''
  );

  return {
    query,
    state,
    versionOptions,
    servers,
    errorMessage,
    retry: () => query.refetch(),
  };
}
