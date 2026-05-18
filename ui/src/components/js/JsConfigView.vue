<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  NCollapse,
  NCollapseItem,
  NFlex,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  NText,
  NTag,
  NAlert,
  NButton,
  useDialog,
  useMessage,
} from 'naive-ui';
import {
  getSdApiV2JsConfigs,
  getSdApiV2JsDeadConfigs,
  postSdApiV2JsConfigs,
  postSdApiV2JsConfigsReset,
  postSdApiV2JsDeadConfigsDelete,
  type ApiPluginConfig,
  type ConfigItem,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';

const emit = defineEmits<{
  dirtyChange: [value: boolean];
}>();

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

const configsQuery = useQuery({
  queryKey: ['js-configs'],
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2JsConfigs({ throwOnError: true });
    return data.item;
  },
});

const deadConfigsQuery = useQuery({
  queryKey: ['js-dead-configs'],
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

const editedValues = ref<Record<string, Record<string, unknown>>>({});
const saveAllLoading = ref(false);

const hasEdits = computed(() =>
  Object.values(editedValues.value).some(item => Object.keys(item).length > 0),
);

watch(
  hasEdits,
  value => {
    emit('dirtyChange', value);
  },
  { immediate: true },
);

const resetMutation = useMutation({
  mutationFn: async (payload: { name: string; keys: string[] }) => {
    await postSdApiV2JsConfigsReset({
      body: { body: payload },
      throwOnError: true,
    });
  },
  onSuccess: async () => {
    message.success('已重置');
    await queryClient.invalidateQueries({ queryKey: ['js-configs'] });
  },
  onError: () => message.error('重置失败'),
});

const deleteDeadMutation = useMutation({
  mutationFn: async (names: string[]) => {
    await postSdApiV2JsDeadConfigsDelete({
      body: { body: { names } },
      throwOnError: true,
    });
  },
  onSuccess: async () => {
    message.success('已删除');
    await queryClient.invalidateQueries({ queryKey: ['js-dead-configs'] });
    await queryClient.invalidateQueries({ queryKey: ['js-configs'] });
  },
  onError: () => message.error('删除失败'),
});

function setEdited(pluginName: string, key: string, value: unknown) {
  if (!editedValues.value[pluginName]) {
    editedValues.value[pluginName] = {};
  }
  editedValues.value[pluginName][key] = value;
}

function getItemType(item: ConfigItem): string {
  return item.type ?? 'string';
}

function getItemVal(item: ConfigItem): unknown {
  return item.value ?? item.defaultValue;
}

function handleDeleteDead(deadList: { name: string }[]) {
  dialog.warning({
    title: '删除死配置',
    content: `确认删除 ${deadList.length} 个死配置？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteDeadMutation.mutateAsync(deadList.map(item => item.name));
    },
  });
}

async function saveAll() {
  if (!hasEdits.value) {
    message.info('无变更');
    return;
  }
  saveAllLoading.value = true;
  try {
    for (const [name, config] of Object.entries(editedValues.value)) {
      if (!Object.keys(config).length) continue;
      await postSdApiV2JsConfigs({
        body: {
          body: {
            name,
            config,
          },
        },
        throwOnError: true,
      });
    }
    editedValues.value = {};
    await queryClient.invalidateQueries({ queryKey: ['js-configs'] });
    message.success('已保存');
  } catch {
    message.error('保存失败');
  } finally {
    saveAllLoading.value = false;
  }
}

defineExpose({
  saveAll,
  saveAllLoading,
});
</script>

<template>
  <div>
    <n-collapse v-if="configItems.length" class="js-config-main">
      <n-collapse-item
        v-for="cfg in configItems"
        :key="cfg.pluginName"
        :title="cfg.pluginName"
        :name="cfg.pluginName"
      >
        <div v-for="item in cfg.items" :key="item.key" class="config-row">
          <n-flex align="center" justify="space-between" wrap>
            <n-flex align="center" size="small">
              <n-text>{{ item.key }}</n-text>
              <n-tag v-if="item.deprecated" size="small" type="error" :bordered="false">
                废弃
              </n-tag>
            </n-flex>

            <n-switch
              v-if="getItemType(item) === 'bool' || getItemType(item) === 'boolean'"
              size="small"
              :value="!!getItemVal(item)"
              @update:value="(v: boolean) => { item.value = v; setEdited(cfg.pluginName, item.key, v); }"
            />

            <n-input-number
              v-else-if="getItemType(item) === 'int' || getItemType(item) === 'float' || getItemType(item) === 'number'"
              size="small"
              :value="Number(getItemVal(item)) || 0"
              class="w-40"
              @update:value="(v: number | null) => { item.value = v; setEdited(cfg.pluginName, item.key, v); }"
            />

            <n-select
              v-else-if="getItemType(item) === 'option' && item.option && Array.isArray(item.option)"
              size="small"
              :value="String(getItemVal(item))"
              :options="(item.option as string[]).map((option: string) => ({ label: String(option), value: String(option) }))"
              class="w-40"
              @update:value="(v: string) => { item.value = v; setEdited(cfg.pluginName, item.key, v); }"
            />

            <n-input
              v-else
              size="small"
              :value="String(getItemVal(item) ?? '')"
              class="w-80"
              @update:value="(v: string) => { item.value = v; setEdited(cfg.pluginName, item.key, v); }"
            />

            <n-button
              size="tiny"
              secondary
              @click="resetMutation.mutateAsync({ name: cfg.pluginName, keys: [item.key] })"
            >
              重置
            </n-button>
          </n-flex>
        </div>
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
            @click="deleteDeadMutation.mutateAsync([dc.name])"
          >
            删除
          </n-button>
        </n-flex>
      </div>
    </div>
  </div>
</template>

<style scoped>
.js-config-main {
  margin-top: 0.5rem;
}

.config-row {
  padding: 0.5rem 0;
}

.config-row + .config-row {
  border-top: 1px solid var(--sd-border-soft);
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

.w-40 {
  width: 10rem;
}

.w-80 {
  width: 20rem;
}
</style>
