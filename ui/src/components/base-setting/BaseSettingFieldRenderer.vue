<template>
  <SettingRow
    :label="field.label"
    :description="fieldFeedback"
    :layout="getBaseSettingFieldLayout(field)"
    :highlighted="highlighted"
    :data-field-id="field.id"
  >
    <div class="base-setting-field-control">
      <template v-if="field.kind === 'unlock-code'">
        <n-button v-if="!revealUnlockCode" @click="revealUnlockCode = true">查看</n-button>
        <div v-else class="unlock-code-text">.master unlock {{ fieldValue }}</div>
      </template>

      <BaseSettingStringListField
        v-else-if="field.kind === 'string-list' && fieldKey"
        :model-value="(fieldValue as string[]) ?? []"
        @update:model-value="updateFieldValue(fieldKey, $event)"
      />

      <BaseSettingMasterListField
        v-else-if="field.kind === 'master-list' && fieldKey"
        :model-value="(fieldValue as string[]) ?? []"
        @update:model-value="updateFieldValue(fieldKey, $event)"
      />

      <BaseSettingNoticeTargetsField
        v-else-if="field.kind === 'notice-targets' && fieldKey"
        :model-value="(fieldValue as string[]) ?? []"
        @update:model-value="updateFieldValue(fieldKey, $event)"
      />

      <n-switch
        v-else-if="field.kind === 'boolean' && fieldKey"
        :value="Boolean(fieldValue)"
        :disabled="fieldDisabled"
        @update:value="updateBoolean"
      />

      <n-input
        v-else-if="field.kind === 'text' && fieldKey"
        :value="String(fieldValue ?? '')"
        :placeholder="field.placeholder"
        :maxlength="field.maxLength || undefined"
        :show-count="Boolean(field.maxLength)"
        @update:value="updateFieldValue(fieldKey, $event)"
      />

      <div v-if="field.kind === 'text' && field.previewSample" class="name-card-template-preview">
        <span class="name-card-template-preview-label">预览：</span>
        <span class="name-card-template-preview-value" :style="previewStyle">
          {{ field.previewSample }}
        </span>
      </div>

      <n-input
        v-else-if="field.kind === 'password' && fieldKey"
        :value="String(fieldValue ?? '')"
        type="password"
        show-password-on="click"
        :placeholder="field.placeholder"
        @update:value="updateFieldValue(fieldKey, $event)"
      />

      <n-input-number
        v-else-if="field.kind === 'number' && fieldKey"
        :value="numberValueOf(fieldValue)"
        clearable
        @update:value="updateFieldValue(fieldKey, $event)"
      />

      <n-select
        v-else-if="field.kind === 'select' && fieldKey"
        :value="String(fieldValue ?? '')"
        :options="field.options"
        filterable
        clearable
        :tag="field.allowCustomValue"
        @update:value="updateSelect"
      />

      <div v-else-if="field.kind === 'number-pair'" class="number-pair">
        <n-input-number
          :value="numberValueOf(pairValues[0])"
          :min="0"
          :max="Number(pairValues[1] ?? 0)"
          @update:value="updatePair(0, $event)"
        />
        <span class="number-pair-sep">-</span>
        <n-input-number
          :value="numberValueOf(pairValues[1])"
          :min="Number(pairValues[0] ?? 0)"
          @update:value="updatePair(1, $event)"
        />
      </div>

      <BaseSettingExtDefaultsField
        v-else-if="field.kind === 'ext-default-settings' && fieldKey"
        :model-value="extDefaultsValueOf(fieldValue)"
        :initial-items="props.initialModel?.extDefaultSettings ?? []"
        @update:model-value="updateFieldValue(fieldKey, $event)"
      />

      <n-button
        v-else-if="field.kind === 'action'"
        type="primary"
        :loading="busyActionId === field.id"
        @click="handleAction"
      >
        {{ field.label }}
      </n-button>

      <div v-else-if="field.kind === 'upload'" class="upgrade-block">
        <n-checkbox v-model:checked="upgradeConfirmed"> 我已阅读功能描述 </n-checkbox>
        <n-flex v-if="isContainerMode" wrap>
          <n-button type="primary" disabled>
            <template #icon>
              <i-tabler-upload />
            </template>
            上传压缩包
          </n-button>
          <n-text type="warning" class="text-xs"
            >容器模式下固件升级被禁用，请手动拉取最新镜像</n-text
          >
        </n-flex>
        <n-upload
          v-else
          action=""
          :show-file-list="false"
          :custom-request="handleUpload"
          :disabled="!upgradeConfirmed || busyActionId === field.id"
        >
          <n-button
            type="primary"
            :disabled="!upgradeConfirmed"
            :loading="busyActionId === field.id"
          >
            <template #icon>
              <i-tabler-upload />
            </template>
            上传压缩包
          </n-button>
        </n-upload>
      </div>
    </div>
  </SettingRow>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { UploadCustomRequestOptions } from 'naive-ui';
import type {
  BaseSettingFieldModel,
  BaseSettingValueModel,
} from '@/features/baseSetting/viewModel';
import SettingRow from '@/components/settings-panel/SettingRow.vue';
import {
  getBaseSettingFieldFeedback,
  getBaseSettingFieldLayout,
} from '@/features/baseSetting/viewModel';
import BaseSettingExtDefaultsField from './BaseSettingExtDefaultsField.vue';
import BaseSettingMasterListField from './BaseSettingMasterListField.vue';
import BaseSettingNoticeTargetsField from './BaseSettingNoticeTargetsField.vue';
import BaseSettingStringListField from './BaseSettingStringListField.vue';

const props = defineProps<{
  field: BaseSettingFieldModel;
  model: BaseSettingValueModel;
  initialModel?: BaseSettingValueModel | null;
  isContainerMode: boolean;
  busyActionId?: string | null;
  highlighted?: boolean;
  runAction: (fieldId: string, payload?: unknown) => Promise<void> | void;
}>();

const emit = defineEmits<{
  updateField: [key: string, value: unknown];
}>();

const dialog = useDialog();
const revealUnlockCode = ref(false);
const upgradeConfirmed = ref(false);

const fieldKey = computed(() => props.field.key);
const fieldValue = computed(() => {
  if (!fieldKey.value) return undefined;
  return props.model[fieldKey.value as keyof BaseSettingValueModel];
});
const fieldDisabled = computed(() => {
  const condition = props.field.disabledWhen;
  if (!condition) return false;
  return props.model[condition.key] === condition.equals;
});
const fieldFeedback = computed(() =>
  fieldDisabled.value && props.field.disabledHint
    ? props.field.disabledHint
    : getBaseSettingFieldFeedback(props.field)
);
const previewStyle = computed(() => {
  const raw = String(fieldValue.value ?? '').trim();
  const template = raw || props.field.previewFallback || '';
  const fontTag = template.match(/<font\b([^>]*)>/i);
  const attrs = fontTag?.[1] ?? '';
  const color = previewAttr(attrs, 'color');
  const size = previewAttr(attrs, 'size');
  const style: Record<string, string> = {};
  if (/^(#[0-9a-f]{3,8}|[a-z]+)$/i.test(color)) style.color = color;
  if (/^\d+$/.test(size)) style.fontSize = previewFontSize(size);
  return style;
});

function previewAttr(attrs: string, name: string): string {
  const match = attrs.match(new RegExp(`${name}\\s*=\\s*["']?([^"'\\s>]+)`, 'i'));
  return match?.[1] ?? '';
}

function previewFontSize(size: string): string {
  const value = Number.parseInt(size, 10);
  if (value <= 1) return 'x-small';
  if (value === 2) return 'small';
  if (value === 4) return 'large';
  if (value >= 5) return 'x-large';
  return 'medium';
}

const pairValues = computed(() =>
  props.field.keys.map(key => props.model[key as keyof BaseSettingValueModel])
);

// 模板里不写联合类型断言：Prettier 会剥掉外层括号，
// 之后 vue-eslint-parser 会把顶层 `|` 当成 Vue 2 过滤器语法而报错。
// 这里的写法目前靠外层 `?? null` 侥幸合规，统一收口到 script 避免后人踩坑。
const numberValueOf = (value: unknown) => (value as number | null | undefined) ?? null;
const extDefaultsValueOf = (value: unknown) =>
  (value as BaseSettingValueModel['extDefaultSettings']) ?? [];

function updateFieldValue(key: string, value: unknown) {
  emit('updateField', key, value);
}

function updateBoolean(value: boolean) {
  const key = fieldKey.value;
  if (!key) return;
  if (value === false && props.field.confirmMessage) {
    dialog.warning({
      title: `关闭${props.field.label}`,
      content: props.field.confirmMessage,
      positiveText: '确定',
      negativeText: '取消',
      closable: false,
      onPositiveClick: () => updateFieldValue(key, value),
    });
    return;
  }
  updateFieldValue(key, value);
}

function updateSelect(value: unknown) {
  const key = fieldKey.value;
  if (!key) return;
  if (value === 'gm' && !props.model.diceRandomGMEnabled && props.field.confirmMessage) {
    dialog.warning({
      title: '启用国密随机源',
      content: props.field.confirmMessage,
      positiveText: '启用',
      negativeText: '取消',
      closable: false,
      onPositiveClick: () => {
        emit('updateField', 'diceRandomGMEnabled', true);
        updateFieldValue(key, value);
      },
    });
    return;
  }
  updateFieldValue(key, value);
}

function updatePair(index: number, value: number | null) {
  const key = props.field.keys[index];
  if (!key) return;
  // 清空输入时按 0 处理：否则输入框已清空但草稿仍是旧值，保存时"清不掉"。
  const next = value ?? 0;
  updateFieldValue(key, next);
  // 保持 起点 <= 终点 的不变量（与后端钳制逻辑一致），
  // 避免清空一端后另一端越界被后端静默忽略。
  const otherIndex = index === 0 ? 1 : 0;
  const otherKey = props.field.keys[otherIndex];
  const otherValue = numberValueOf(pairValues.value[otherIndex]);
  if (!otherKey || otherValue === null) return;
  if (index === 0 && otherValue < next) updateFieldValue(otherKey, next);
  if (index === 1 && otherValue > next) updateFieldValue(otherKey, next);
}

async function handleAction() {
  await props.runAction(props.field.id);
}

async function handleUpload(options: UploadCustomRequestOptions) {
  try {
    await Promise.resolve(props.runAction(props.field.id, options.file.file as File));
    options.onFinish();
  } catch {
    options.onError();
  }
}
</script>

<style scoped>
.base-setting-field-control {
  width: 100%;
}

.base-setting-field-control :deep(.n-select),
.base-setting-field-control :deep(.n-input),
.base-setting-field-control :deep(.n-input-number) {
  width: 100%;
}

.unlock-code-text {
  font-weight: 700;
}

.name-card-template-preview {
  margin-top: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0.375rem;
  background: var(--n-color-embedded, rgba(127, 127, 127, 0.08));
}

.name-card-template-preview-label {
  margin-right: 0.5rem;
  opacity: 0.65;
}

.number-pair {
  display: flex;
  align-items: center;
  width: min(100%, 24rem);
}

.number-pair-sep {
  margin: 0 1rem;
}

.upgrade-block {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
</style>
