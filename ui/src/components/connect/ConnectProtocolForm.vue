<template>
  <OfficialQQModePanel v-if="protocolModule.formKind === 'officialqq'" v-model="officialQqMode" />

  <n-alert v-if="protocol && !protocol.available" type="warning" :show-icon="false">
    {{ protocol.disabledReason }}
  </n-alert>

  <n-alert v-if="schemasError" type="error" :show-icon="false">
    配置项读取失败，请稍后重试。
  </n-alert>

  <DynamicForm
    :model-value="modelValue"
    :schema="visibleSchema"
    :disabled="submitting || testModeDisabled"
    :label-placement="isMobile ? 'top' : 'left'"
    :label-width="isMobile ? undefined : 108"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #field="{ item, fieldKey, value, setValue }">
      <LagrangeSignInfoField
        v-if="
          protocolModule.formKind === 'sign-info' &&
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
      <n-input
        v-else-if="item.input_type === 0"
        :value="value as string"
        :type="item.sensitive ? 'password' : 'text'"
        :disabled="submitting || testModeDisabled"
        :placeholder="item.placeholder"
        show-password-on="mousedown"
        @update:value="setValue"
      />
      <n-input-number
        v-else-if="item.input_type === 1"
        :value="value as number | null"
        :disabled="submitting || testModeDisabled"
        :min="1"
        :max="65535"
        :placeholder="item.placeholder"
        style="width: 100%"
        @update:value="setValue"
      />
      <n-switch
        v-else-if="item.input_type === 10"
        :value="Boolean(value)"
        :disabled="submitting || testModeDisabled"
        @update:value="setValue"
      />
    </template>
  </DynamicForm>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { SelectOption } from 'naive-ui';
import type { FormConfigItem, ProtocolDefinition } from '@/api';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import { fieldKeyOf, type DynamicFormModel } from '@/components/shared/dynamicFormModel';
import OfficialQQModePanel from './protocol/OfficialQQModePanel.vue';
import LagrangeSignInfoField from './protocol/LagrangeSignInfoField.vue';
import { getConnectProtocolModule } from '@/features/connect/protocols';
import type { OfficialQQMode } from '@/features/connect/officialQQ';
import type { SignInfoState } from '@/features/connect/signInfoState';

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

const protocolModule = computed(() => getConnectProtocolModule(props.protocol?.key ?? ''));

const visibleSchema = computed(() => {
  if (protocolModule.value.formKind !== 'officialqq') return props.schema;

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
