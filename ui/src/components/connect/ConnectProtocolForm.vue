<template>
  <div class="sd-section-flow">
    <OfficialQQModePanel v-if="protocol?.key === 'officialqq'" v-model="officialQqMode" />

    <TipBox v-if="protocol && !protocol.available" type="warning">
      {{ protocol.disabledReason }}
    </TipBox>

    <TipBox v-if="schemasError" type="error"> 配置项读取失败，请稍后重试。 </TipBox>

    <DynamicForm
      :model-value="modelValue"
      :schema="visibleSchema"
      :disabled="submitting || testModeDisabled"
      :label-placement="isMobile ? 'top' : 'left'"
      :label-width="isMobile ? undefined : 108"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #field="{ fieldKey, value, setValue }">
        <LagrangeSignInfoField
          v-if="
            protocol?.key === 'lagrange' &&
            (fieldKey === 'signServerVersion' || fieldKey === 'signServerName')
          "
          :field-key="fieldKey"
          :value="value"
          :sign-info-state="signInfoState"
          :sign-info-error-message="signInfoErrorMessage"
          :sign-version-options="signVersionOptions"
          :sign-servers="signServers"
          @retry="emit('retrySignInfo')"
          @update:value="setValue"
        />
      </template>
    </DynamicForm>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { SelectOption } from 'naive-ui';
import type { FormConfigItem, ProtocolDefinition } from '@/api';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import { fieldKeyOf, type DynamicFormModel } from '@/components/shared/dynamicFormModel';
import OfficialQQModePanel from './protocol/OfficialQQModePanel.vue';
import LagrangeSignInfoField from './protocol/LagrangeSignInfoField.vue';
import type { OfficialQQMode } from '@/features/connect/officialQQ';
import type { SignInfoState } from '@/features/connect/signInfoState';
import TipBox from '@/components/shared/TipBox.vue';

const props = defineProps<{
  protocol: ProtocolDefinition | null;
  schema: FormConfigItem[];
  modelValue: DynamicFormModel;
  schemasError: boolean;
  signInfoState: SignInfoState;
  signInfoErrorMessage: string;
  signVersionOptions: SelectOption[];
  signServers: SelectOption[];
  isMobile: boolean;
  submitting: boolean;
  officialQqMode: OfficialQQMode;
  testModeDisabled: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: DynamicFormModel];
  'update:officialQqMode': [value: OfficialQQMode];
  retrySignInfo: [];
}>();

const officialQqMode = computed({
  get: () => props.officialQqMode,
  set: value => emit('update:officialQqMode', value),
});

const visibleSchema = computed(() => {
  if (props.protocol?.key !== 'officialqq') return props.schema;

  const hiddenFields = new Set<string>();
  if (!props.modelValue.useWebhook) {
    hiddenFields.add('webhookPath');
    hiddenFields.add('webhookPort');
  }
  if (officialQqMode.value === 'qrcode') {
    hiddenFields.add('appID');
    hiddenFields.add('appSecret');
  }

  return props.schema.filter(item => !hiddenFields.has(fieldKeyOf(item)));
});
</script>
