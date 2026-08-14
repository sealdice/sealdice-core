<template>
  <n-drawer
    :show="show"
    :width="drawerWidth"
    placement="right"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content :title="packageTitle" closable>
      <n-spin :show="loading">
        <n-space vertical size="large">
          <n-descriptions bordered label-placement="left" :column="1">
            <n-descriptions-item label="包 ID">{{
              pkg?.manifest?.package?.id || '-'
            }}</n-descriptions-item>
            <n-descriptions-item label="版本">{{
              pkg?.manifest?.package?.version || '-'
            }}</n-descriptions-item>
            <n-descriptions-item label="状态">{{ pkg?.state || '-' }}</n-descriptions-item>
            <n-descriptions-item label="作者">{{
              pkg?.manifest?.package?.authors?.join(' / ') || '-'
            }}</n-descriptions-item>
            <n-descriptions-item label="描述">{{
              pkg?.manifest?.package?.description || '暂无描述'
            }}</n-descriptions-item>
            <n-descriptions-item label="安装路径">{{
              pkg?.installPath || '-'
            }}</n-descriptions-item>
            <n-descriptions-item label="源文件">{{ pkg?.sourcePath || '-' }}</n-descriptions-item>
          </n-descriptions>

          <n-card title="配置" size="small">
            <n-space v-if="schemaEntries.length === 0" vertical>
              <n-text depth="3">该扩展包没有可编辑配置。</n-text>
            </n-space>
            <n-form v-else label-placement="top">
              <n-form-item
                v-for="[fieldKey, fieldSchema] in schemaEntries"
                :key="fieldKey"
                :label="fieldSchema.title || fieldKey"
                :validation-status="jsonFieldErrors[fieldKey] ? 'error' : undefined"
                :feedback="jsonFieldErrors[fieldKey]"
              >
                <n-switch
                  v-if="fieldSchema.type === 'boolean'"
                  :value="Boolean(draft[fieldKey])"
                  @update:value="(value: boolean) => updateValue(fieldKey, value)"
                />
                <n-input-number
                  v-else-if="fieldSchema.type === 'number' || fieldSchema.type === 'integer'"
                  :value="numberValueOf(fieldKey)"
                  class="w-full"
                  @update:value="
                    (value: number | null) =>
                      updateValue(fieldKey, value ?? fieldSchema.default ?? 0)
                  "
                />
                <n-select
                  v-else-if="Array.isArray(fieldSchema.enum) && fieldSchema.enum.length > 0"
                  :value="selectValueOf(fieldKey)"
                  :options="enumOptions(fieldSchema.enum)"
                  @update:value="(value: SelectValue | null) => updateValue(fieldKey, value)"
                />
                <n-input
                  v-else-if="fieldSchema.type === 'string'"
                  :value="String(draft[fieldKey] ?? '')"
                  :type="fieldSchema.secret ? 'password' : 'text'"
                  show-password-on="click"
                  @update:value="(value: string) => updateValue(fieldKey, value)"
                />
                <n-input
                  v-else
                  :value="jsonValueOf(fieldKey)"
                  class="sd-code-text"
                  type="textarea"
                  :autosize="{ minRows: 3, maxRows: 8 }"
                  @update:value="(value: string) => updateJsonValue(fieldKey, value)"
                />
                <template v-if="fieldSchema.description">
                  <n-text depth="3">{{ fieldSchema.description }}</n-text>
                </template>
              </n-form-item>
            </n-form>

            <template #footer>
              <n-space justify="end">
                <n-button
                  :loading="saving"
                  :disabled="hasJsonErrors"
                  type="primary"
                  @click="emit('save-config', { ...draft })"
                >
                  保存配置
                </n-button>
              </n-space>
            </template>
          </n-card>

          <n-card title="文件树" size="small">
            <PackageFileTree :files="pkg?.files" />
          </n-card>
        </n-space>
      </n-spin>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue';
import type { SelectOption } from 'naive-ui';
import type { Instance } from '@/api';
import { useResponsiveOverlayWidth } from '@/features/responsive/useResponsiveOverlayWidth';
import PackageFileTree from './PackageFileTree.vue';

type ConfigFieldSchema = {
  type?: string;
  title?: string;
  description?: string;
  default?: unknown;
  secret?: boolean;
  enum?: unknown[] | null;
};

type SelectValue = string | number;

const props = defineProps<{
  show: boolean;
  pkg: Instance | null;
  config: Record<string, unknown> | null;
  schema: Record<string, ConfigFieldSchema> | null;
  loading?: boolean;
  saving?: boolean;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'save-config': [value: Record<string, unknown>];
}>();

const draft = reactive<Record<string, unknown>>({});
const jsonFieldErrors = reactive<Record<string, string>>({});
const { width: drawerWidth } = useResponsiveOverlayWidth({ maxWidth: 720, gutter: 16 });

const packageTitle = computed(() => props.pkg?.manifest?.package?.name || '扩展包详情');
const schemaEntries = computed(() => Object.entries(props.schema ?? {}));

watch(
  () => [props.show, props.config, props.schema] as const,
  () => {
    for (const key of Object.keys(draft)) {
      delete draft[key];
    }
    for (const key of Object.keys(jsonFieldErrors)) {
      delete jsonFieldErrors[key];
    }
    const config = props.config ?? {};
    for (const [fieldKey, fieldSchema] of schemaEntries.value) {
      draft[fieldKey] = config[fieldKey] ?? fieldSchema.default ?? defaultValueOf(fieldSchema.type);
    }
  },
  { immediate: true }
);

const hasJsonErrors = computed(() => Object.keys(jsonFieldErrors).length > 0);

function defaultValueOf(type?: string) {
  switch (type) {
    case 'boolean':
      return false;
    case 'number':
    case 'integer':
      return 0;
    case 'string':
      return '';
    default:
      return null;
  }
}

function updateValue(fieldKey: string, value: unknown) {
  draft[fieldKey] = value;
}

function updateJsonValue(fieldKey: string, value: string) {
  try {
    draft[fieldKey] = value.trim() ? JSON.parse(value) : null;
    delete jsonFieldErrors[fieldKey];
  } catch {
    draft[fieldKey] = value;
    jsonFieldErrors[fieldKey] = '请输入有效 JSON';
  }
}

function enumOptions(values: readonly unknown[]): SelectOption[] {
  return values.map(value => ({
    label: String(value),
    value: value as SelectValue,
  }));
}

function selectValueOf(fieldKey: string) {
  const value = draft[fieldKey];
  return typeof value === 'string' || typeof value === 'number' ? value : null;
}

function numberValueOf(fieldKey: string) {
  const value = draft[fieldKey];
  return typeof value === 'number' ? value : Number(value ?? 0);
}

function jsonValueOf(fieldKey: string) {
  const value = draft[fieldKey];
  if (typeof value === 'string') return value;
  return JSON.stringify(value ?? null, null, 2);
}
</script>
