<template>
  <div>
    <n-collapse v-if="configItems.length" class="js-config-main">
      <n-collapse-item
        v-for="cfg in configItems"
        :key="cfg.pluginName"
        :title="cfg.pluginName"
        :name="cfg.pluginName"
      >
        <ResponsiveTabs
          v-if="cfg.groups.length"
          :value="
            cfg.groups.some(group => group.name === activeGroups[cfg.pluginName])
              ? activeGroups[cfg.pluginName]
              : cfg.groups[0]?.name
          "
          :options="cfg.groups.map(group => ({ label: group.name, value: group.name }))"
          :compact-at="600"
          @update:value="activeGroups[cfg.pluginName] = $event"
        >
          <template #panel="{ option }">
            <JsConfigItemEditor
              v-for="item in cfg.groups.find(group => group.name === option.value)?.items ?? []"
              :key="item.key"
              :item="item"
              :plugin-name="cfg.pluginName"
              :error-text="configErrors[buildConfigErrorKey(cfg.pluginName, item.key)]"
              :checking="checkingKeys[buildConfigErrorKey(cfg.pluginName, item.key)]"
              :disabled="
                isTestMode || resetMutation.isPending.value || deleteDeadMutation.isPending.value
              "
              :reset-disabled="writesBlocked"
              @change="setEdited"
              @reset="resetSingleConfig"
            />
          </template>
        </ResponsiveTabs>
        <n-text v-if="!cfg.groups.length" depth="3">无配置项</n-text>
      </n-collapse-item>
    </n-collapse>

    <n-text v-else depth="3">暂无活跃配置</n-text>

    <div v-if="deadConfigsQuery.data.value?.length" class="dead-configs-block">
      <n-flex size="small" align="center">
        <TipBox type="warning">
          以下是残留的死配置：对应的插件当前未加载（可能已被删除、禁用或加载失败）。删除会清除该插件名下的全部配置且不可恢复，请确认无需保留。
        </TipBox>
        <n-button
          type="error"
          size="small"
          :disabled="writesBlocked"
          @click="handleDeleteDead(deadConfigsQuery.data.value ?? [])"
        >
          删除全部
        </n-button>
      </n-flex>

      <div v-for="dc in deadConfigsQuery.data.value" :key="dc.name" class="dead-config-row">
        <n-flex align="center" justify="space-between">
          <n-text>{{ dc.name }}</n-text>
          <n-button
            :disabled="writesBlocked"
            size="tiny"
            type="error"
            secondary
            @click="handleDeleteDead([dc])"
          >
            删除
          </n-button>
        </n-flex>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue';
import { cloneDeep } from 'es-toolkit/compat';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { NCollapse, NCollapseItem, NFlex, NText, NButton, useDialog, useMessage } from 'naive-ui';
import { storeToRefs } from 'pinia';
import {
  getSdApiV2JsConfigsOptions,
  getSdApiV2JsConfigsQueryKey,
  getSdApiV2JsDeadConfigsOptions,
  getSdApiV2JsDeadConfigsQueryKey,
  postSdApiV2JsConfigs,
  postSdApiV2JsConfigsReset,
  postSdApiV2JsDeadConfigsDelete,
  postSdApiV2JsCronCheck,
  type GetSdApiV2JsConfigsResponse,
} from '@/api';
import JsConfigItemEditor from './JsConfigItemEditor.vue';
import ResponsiveTabs from '@/components/shared/ResponsiveTabs.vue';
import TipBox from '@/components/shared/TipBox.vue';
import {
  buildConfigErrorKey,
  groupPluginConfigItems,
  isDailyTaskExpressionValid,
  setConfigError,
  shouldBlockConfigSave,
  type JsConfigErrorMap,
} from '@/features/js/configModel';
import { getErrorMessage } from '@/features/auth/error';
import { useAuthStore } from '@/features/auth/store';
import { useTestMode } from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';
import { useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const { hasAccessToken } = storeToRefs(useAuthStore());
const { isTestMode } = useTestMode();
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
const editedValues = ref<Record<string, Record<string, unknown>>>(Object.create(null));
const activeGroups = reactive<Record<string, string>>(Object.create(null));
const configErrors = reactive<JsConfigErrorMap>(Object.create(null));
const checkingKeys = ref<Record<string, boolean>>(Object.create(null));
const validations = new Map<string, AbortController>();
const configItems = computed(() =>
  Object.entries(configsQuery.data.value ?? {}).map(([pluginName, plugin]) => ({
    pluginName,
    groups: groupPluginConfigItems(
      (plugin?.configs ?? []).map(item => ({
        ...item,
        value: Object.prototype.hasOwnProperty.call(editedValues.value[pluginName] ?? {}, item.key)
          ? editedValues.value[pluginName]![item.key]
          : item.value,
      }))
    ),
  }))
);
const hasEdits = computed(() =>
  Object.values(editedValues.value).some(values => Object.keys(values).length > 0)
);
const canSave = computed(
  () =>
    hasEdits.value &&
    !isTestMode.value &&
    !shouldBlockConfigSave(configErrors) &&
    Object.keys(checkingKeys.value).length === 0 &&
    !resetMutation.isPending.value &&
    !deleteDeadMutation.isPending.value
);
const writesBlocked = computed(
  () =>
    isTestMode.value ||
    saveMutation.isPending.value ||
    resetMutation.isPending.value ||
    deleteDeadMutation.isPending.value
);

function clearFieldValidation(pluginName: string, key: string) {
  const errorKey = buildConfigErrorKey(pluginName, key);
  validations.get(errorKey)?.abort();
  validations.delete(errorKey);
  delete checkingKeys.value[errorKey];
  setConfigError(configErrors, pluginName, key, '');
}

function setEdited(pluginName: string, key: string, value: unknown) {
  const item = configsQuery.data.value?.[pluginName]?.configs?.find(item => item.key === key);
  if (!item) return;
  clearFieldValidation(pluginName, key);
  if (JSON.stringify(value) === JSON.stringify(item.value ?? item.defaultValue)) {
    delete editedValues.value[pluginName]?.[key];
    return;
  }
  editedValues.value[pluginName] = {
    ...editedValues.value[pluginName],
    [key]: cloneDeep(value),
  };
  void validateConfigValue(pluginName, key, String(value ?? ''), item.type ?? '');
}
async function validateConfigValue(pluginName: string, key: string, value: string, type: string) {
  if (type === 'task:daily') {
    setConfigError(
      configErrors,
      pluginName,
      key,
      isDailyTaskExpressionValid(value) ? '' : '每日定时任务格式错误，应为 HH:mm'
    );
    return;
  }
  if (type !== 'task:cron') return;
  const errorKey = buildConfigErrorKey(pluginName, key);
  const controller = new AbortController();
  validations.set(errorKey, controller);
  checkingKeys.value[errorKey] = true;
  try {
    await postSdApiV2JsCronCheck({
      body: { expr: value },
      signal: controller.signal,
      throwOnError: true,
    });
    if (validations.get(errorKey) === controller) setConfigError(configErrors, pluginName, key, '');
  } catch (error) {
    if (validations.get(errorKey) === controller && !controller.signal.aborted)
      setConfigError(configErrors, pluginName, key, getErrorMessage(error, 'Cron 表达式格式错误'));
  } finally {
    if (validations.get(errorKey) === controller) {
      validations.delete(errorKey);
      delete checkingKeys.value[errorKey];
    }
  }
}
function clearValidations() {
  validations.forEach(controller => controller.abort());
  validations.clear();
  checkingKeys.value = Object.create(null);
  Object.keys(configErrors).forEach(key => delete configErrors[key]);
}
onBeforeUnmount(clearValidations);

function commitConfigSnapshot(name: string, config: Record<string, unknown>) {
  queryClient.setQueryData<GetSdApiV2JsConfigsResponse>(getSdApiV2JsConfigsQueryKey(), previous => {
    const plugin = previous?.item[name];
    if (!previous || !plugin) return previous;
    return {
      ...previous,
      item: {
        ...previous.item,
        [name]: {
          ...plugin,
          configs: (plugin.configs ?? []).map(item =>
            Object.prototype.hasOwnProperty.call(config, item.key)
              ? { ...item, value: cloneDeep(config[item.key]) }
              : item
          ),
        },
      },
    };
  });
}

const saveMutation = useMutation({
  mutationFn: async () => {
    const snapshot = cloneDeep(editedValues.value);
    try {
      for (const [name, config] of Object.entries(snapshot)) {
        if (!Object.keys(config).length) continue;
        const { data } = await postSdApiV2JsConfigs({ body: { name, config }, throwOnError: true });
        if (!data.item.success) throw new Error(`插件「${name}」配置未保存`);
        // 每个插件是一个后端提交单元；先确认已经成功的快照，再处理下一个插件。
        commitConfigSnapshot(name, config);
        for (const [key, value] of Object.entries(config)) {
          if (
            Object.prototype.hasOwnProperty.call(editedValues.value[name] ?? {}, key) &&
            JSON.stringify(editedValues.value[name]![key]) === JSON.stringify(value)
          ) {
            delete editedValues.value[name]![key];
          }
        }
      }
    } finally {
      await queryClient.invalidateQueries({ queryKey: getSdApiV2JsConfigsQueryKey() });
    }
  },
  onSuccess: () => message.success('已保存'),
  onError: error => notifyError(error, message, '保存失败'),
});
const resetMutation = useMutation({
  mutationFn: async ({ name, key }: { name: string; key: string }) => {
    const defaultValue = cloneDeep(
      configsQuery.data.value?.[name]?.configs?.find(item => item.key === key)?.defaultValue
    );
    const { data } = await postSdApiV2JsConfigsReset({
      body: { name, keys: [key] },
      throwOnError: true,
    });
    if (!data.item.success) throw new Error('配置未重置');
    return { name, key, defaultValue };
  },
  onSuccess: async ({ name, key, defaultValue }) => {
    commitConfigSnapshot(name, { [key]: defaultValue });
    delete editedValues.value[name]?.[key];
    clearFieldValidation(name, key);
    message.success('已重置');
    await queryClient.invalidateQueries({ queryKey: getSdApiV2JsConfigsQueryKey() });
  },
  onError: error => notifyError(error, message, '重置失败'),
});
function resetSingleConfig(name: string, key: string) {
  if (writesBlocked.value) return;
  resetMutation.mutate({ name, key });
}
const deleteDeadMutation = useMutation({
  mutationFn: async (names: string[]) => {
    const { data } = await postSdApiV2JsDeadConfigsDelete({ body: { names }, throwOnError: true });
    if (!data.item.success) throw new Error('配置未删除');
  },
  onSuccess: async (_, names) => {
    for (const name of names) {
      Object.keys(editedValues.value[name] ?? {}).forEach(key => clearFieldValidation(name, key));
      delete editedValues.value[name];
    }
    message.success('已删除');
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: getSdApiV2JsDeadConfigsQueryKey() }),
      queryClient.invalidateQueries({ queryKey: getSdApiV2JsConfigsQueryKey() }),
    ]);
  },
  onError: error => notifyError(error, message, '删除失败'),
});
function handleDeleteDead(deadList: { name: string }[]) {
  if (writesBlocked.value) return;
  dialog.warning({
    title: '删除死配置',
    content:
      deadList.length === 1
        ? `确认删除插件「${deadList[0]!.name}」的全部配置？此操作不可恢复。`
        : `确认删除 ${deadList.length} 个插件的全部配置？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => {
      if (!writesBlocked.value)
        return deleteDeadMutation.mutateAsync(deadList.map(item => item.name));
    },
  });
}
useUnsavedChanges('js-config', {
  label: 'JS 配置',
  dirty: hasEdits,
  saving: computed(() => saveMutation.isPending.value),
  canSave,
  save: () => {
    if (!canSave.value || saveMutation.isPending.value) return;
    return saveMutation.mutateAsync();
  },
  discard: () => {
    editedValues.value = Object.create(null);
    clearValidations();
  },
  confirmMessage: 'JS 配置还有修改，确定要忽略？',
});
</script>

<style scoped>
.js-config-main {
  margin-top: 0.5rem;
}

.dead-configs-block {
  margin-top: 1rem;
}

.dead-config-row {
  padding: 0.35rem 1rem;
}

.dead-config-row + .dead-config-row {
  border-top: 1px solid var(--sd-border-soft);
}

@media screen and (max-width: 639.9px) {
}
</style>
