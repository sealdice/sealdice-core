<template>
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

    <n-switch
      v-else-if="field.kind === 'boolean' && fieldKey"
      :value="Boolean(fieldValue)"
      @update:value="updateBoolean"
    />

    <n-input
      v-else-if="field.kind === 'text' && fieldKey"
      :value="String(fieldValue ?? '')"
      :placeholder="field.placeholder"
      @update:value="updateFieldValue(fieldKey, $event)"
    />

    <n-input
      v-else-if="field.kind === 'password' && fieldKey"
      :value="String(fieldValue ?? '')"
      type="password"
      show-password-on="mousedown"
      :placeholder="field.placeholder"
      @update:value="updateFieldValue(fieldKey, $event)"
    />

    <n-input-number
      v-else-if="field.kind === 'number' && fieldKey"
      :value="(fieldValue as number | null | undefined) ?? null"
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
      @update:value="updateFieldValue(fieldKey, $event)"
    />

    <div v-else-if="field.kind === 'number-pair'" class="number-pair">
      <n-input-number
        :value="(pairValues[0] as number | null | undefined) ?? null"
        :min="0"
        :max="Number(pairValues[1] ?? 0)"
        @update:value="updatePair(0, $event)"
      />
      <span class="number-pair-sep">-</span>
      <n-input-number
        :value="(pairValues[1] as number | null | undefined) ?? null"
        :min="Number(pairValues[0] ?? 0)"
        @update:value="updatePair(1, $event)"
      />
    </div>

    <BaseSettingExtDefaultsField
      v-else-if="field.kind === 'ext-default-settings' && fieldKey"
      :model-value="(fieldValue as BaseSettingValueModel['extDefaultSettings']) ?? []"
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
      <n-checkbox v-model:checked="upgradeConfirmed">
        我已阅读功能描述
      </n-checkbox>
      <n-flex v-if="isContainerMode" wrap>
        <n-tooltip>
          <template #trigger>
            <n-button type="primary" disabled>
              <template #icon>
                <i-carbon-upload />
              </template>
              上传压缩包
            </n-button>
          </template>
          容器模式下固件升级被禁用，请手动拉取最新镜像
        </n-tooltip>
        <n-text type="warning" class="text-xs">容器模式下固件升级被禁用，请手动拉取最新镜像</n-text>
      </n-flex>
      <n-upload
        v-else
        action=""
        :show-file-list="false"
        :custom-request="handleUpload"
        :disabled="!upgradeConfirmed || busyActionId === field.id"
      >
        <n-button type="primary" :disabled="!upgradeConfirmed" :loading="busyActionId === field.id">
          <template #icon>
            <i-carbon-upload />
          </template>
          上传压缩包
        </n-button>
      </n-upload>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { UploadCustomRequestOptions } from 'naive-ui';
import type { BaseSettingFieldModel, BaseSettingValueModel } from '@/features/baseSetting/viewModel';
import BaseSettingExtDefaultsField from './BaseSettingExtDefaultsField.vue';
import BaseSettingStringListField from './BaseSettingStringListField.vue';

const props = defineProps<{
  field: BaseSettingFieldModel;
  model: BaseSettingValueModel;
  initialModel?: BaseSettingValueModel | null;
  isContainerMode: boolean;
  busyActionId?: string | null;
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

const pairValues = computed(() => props.field.keys.map(key => props.model[key as keyof BaseSettingValueModel]));

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

function updatePair(index: number, value: number | null) {
  const key = props.field.keys[index];
  if (!key || value === null) return;
  updateFieldValue(key, value);
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
