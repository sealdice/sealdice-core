<script setup lang="ts">
import type { BaseSettingFieldModel, BaseSettingValueModel } from '@/features/baseSetting/viewModel';
import {
  getBaseSettingFieldFeedback,
  getBaseSettingFieldLayout,
} from '@/features/baseSetting/viewModel';
import SettingRow from '@/components/settings-panel/SettingRow.vue';
import BaseSettingFieldControl from './BaseSettingFieldControl.vue';

defineProps<{
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
</script>

<template>
  <SettingRow
    :label="field.label"
    :description="getBaseSettingFieldFeedback(field)"
    :layout="getBaseSettingFieldLayout(field)"
    :highlighted="highlighted"
    :data-field-id="field.id"
  >
    <BaseSettingFieldControl
      :field="field"
      :model="model"
      :initial-model="initialModel"
      :is-container-mode="isContainerMode"
      :busy-action-id="busyActionId"
      :run-action="runAction"
      @update-field="(key, value) => emit('updateField', key, value)"
    />
  </SettingRow>
</template>
