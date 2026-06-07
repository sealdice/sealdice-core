<template>
  <div>
    <n-collapse v-if="configItems.length" class="js-config-main">
      <n-collapse-item
        v-for="cfg in configItems"
        :key="cfg.pluginName"
        :title="cfg.pluginName"
        :name="cfg.pluginName"
      >
        <n-tabs v-if="cfg.items.length" type="line" size="small" animated>
          <n-tab-pane
            v-for="group in getPluginGroups(cfg.items)"
            :key="`${cfg.pluginName}-${group.name}`"
            :name="group.name"
            :tab="group.name"
          >
            <JsConfigItemEditor
              v-for="item in group.items"
              :key="item.key"
              :item="item"
              :plugin-name="cfg.pluginName"
              :error-text="configErrors[buildConfigErrorKey(cfg.pluginName, item.key)]"
              :checking="!!checkingKeys[buildConfigErrorKey(cfg.pluginName, item.key)]"
              @change="setEdited"
              @reset="resetSingleConfig"
              @validate="validateConfigValue"
            />
          </n-tab-pane>
        </n-tabs>
        <n-text v-if="!cfg.items.length" depth="3">无配置项</n-text>
      </n-collapse-item>
    </n-collapse>

    <n-text v-else depth="3">暂无活跃配置</n-text>

    <div v-if="deadConfigsQuery.data.value?.length" class="dead-configs-block">
      <n-flex size="small" align="center">
        <n-alert type="warning" :show-icon="false">
          <template #header>
            以下是残留的死配置（插件已不存在但配置仍在），可安全删除
          </template>
        </n-alert>
        <n-button
          type="error"
          size="small"
          :disabled="!deadConfigsQuery.data.value?.length"
          @click="handleDeleteDead(deadConfigsQuery.data.value ?? [])"
        >
          删除全部
        </n-button>
      </n-flex>

      <div v-for="dc in deadConfigsQuery.data.value" :key="dc.name" class="dead-config-row">
        <n-flex align="center" justify="space-between">
          <n-text>{{ dc.name }}</n-text>
          <n-button
            size="tiny"
            type="error"
            secondary
            @click="deleteDeadConfigs([dc.name])"
          >
            删除
          </n-button>
        </n-flex>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import {
  NCollapse,
  NCollapseItem,
  NFlex,
  NText,
  NAlert,
  NButton,
  useDialog,
  useMessage,
} from 'naive-ui';
import { postSdApiV2JsCronCheck } from '@/api';
import JsConfigItemEditor from '@/components/js/JsConfigItemEditor.vue';
import {
  buildConfigErrorKey,
  groupPluginConfigItems,
  isDailyTaskExpressionValid,
  setConfigError,
  shouldBlockConfigSave,
  type JsConfigErrorMap,
} from '@/features/js/configModel';
import { getErrorMessage } from '@/features/auth/error';
import { useJsConfig } from '@/features/js/useJsConfig';

const emit = defineEmits<{
  dirtyChange: [value: boolean];
}>();

const message = useMessage();
const dialog = useDialog();
const {
  deadConfigsQuery,
  configItems,
  resetMutation,
  deleteDeadMutation,
  savePluginConfigs,
} = useJsConfig();

const editedValues = ref<Record<string, Record<string, unknown>>>({});
const configErrors = reactive<JsConfigErrorMap>({});
const checkingKeys = ref<Record<string, boolean>>({});
const saveAllLoading = ref(false);

const hasEdits = computed(() =>
  Object.values(editedValues.value).some(item => Object.keys(item).length > 0),
);
const hasConfigErrors = computed(() => shouldBlockConfigSave(configErrors));

watch(
  hasEdits,
  value => {
    emit('dirtyChange', value);
  },
  { immediate: true },
);

function setEdited(pluginName: string, key: string, value: unknown) {
  if (!editedValues.value[pluginName]) {
    editedValues.value[pluginName] = {};
  }
  editedValues.value[pluginName][key] = value;
}

function getPluginGroups(items: Parameters<typeof groupPluginConfigItems>[0]) {
  return groupPluginConfigItems(items);
}

function handleDeleteDead(deadList: { name: string }[]) {
  dialog.warning({
    title: '删除死配置',
    content: `确认删除 ${deadList.length} 个死配置？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteDeadConfigs(deadList.map(item => item.name));
    },
  });
}

async function resetConfigItem(payload: { name: string; keys: string[] }) {
  try {
    await resetMutation.mutateAsync(payload);
    message.success('已重置');
  } catch {
    message.error('重置失败');
  }
}

async function resetSingleConfig(pluginName: string, key: string) {
  await resetConfigItem({ name: pluginName, keys: [key] });
}

async function deleteDeadConfigs(names: string[]) {
  try {
    await deleteDeadMutation.mutateAsync(names);
    message.success('已删除');
  } catch {
    message.error('删除失败');
  }
}

async function saveAll() {
  if (!hasEdits.value) {
    message.info('无变更');
    return;
  }
  if (hasConfigErrors.value) {
    message.error('配置格式错误，请修正后再保存');
    return;
  }
  saveAllLoading.value = true;
  try {
    await savePluginConfigs(editedValues.value);
    editedValues.value = {};
    message.success('已保存');
  } catch {
    message.error('保存失败');
  } finally {
    saveAllLoading.value = false;
  }
}

async function validateConfigValue(pluginName: string, key: string, value: string, type: string) {
  if (type === 'task:daily') {
    setConfigError(
      configErrors,
      pluginName,
      key,
      isDailyTaskExpressionValid(value) ? '' : '每日定时任务格式错误，应为 HH:mm',
    );
    return;
  }
  if (type !== 'task:cron') return;

  const errorKey = buildConfigErrorKey(pluginName, key);
  checkingKeys.value = { ...checkingKeys.value, [errorKey]: true };
  try {
    await postSdApiV2JsCronCheck({
      body: { expr: value },
      throwOnError: true,
    });
    setConfigError(configErrors, pluginName, key, '');
  } catch (error) {
    setConfigError(configErrors, pluginName, key, getErrorMessage(error, 'Cron 表达式格式错误'));
  } finally {
    const nextChecking = { ...checkingKeys.value };
    delete nextChecking[errorKey];
    checkingKeys.value = nextChecking;
  }
}

defineExpose({
  saveAll,
  saveAllLoading,
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
