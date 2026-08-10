<template>
  <OfficialQQModePanel v-if="protocolModule.formKind === 'officialqq'" v-model="officialQQMode" />

  <n-alert v-if="protocol && !protocol.available" type="warning" :show-icon="false">
    {{ protocol.disabledReason }}
  </n-alert>

  <n-alert v-if="schemasError" type="error" :show-icon="false">
    配置项读取失败，请稍后重试。
  </n-alert>

  <DynamicForm
    :model-value="modelValue"
    :schema="schema"
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
        :disabled="isOfficialCredentialDisabled(fieldKey) || submitting || testModeDisabled"
        :placeholder="item.placeholder"
        show-password-on="mousedown"
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
import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
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

const officialQQMode = computed({
  get: () => props.officialQqMode,
  set: value => emit('update:officialQqMode', value),
});

const protocolModule = computed(() => getConnectProtocolModule(props.protocol?.key ?? ''));

const isOfficialCredentialDisabled = (fieldKey: string) =>
  protocolModule.value.formKind === 'officialqq' &&
  officialQQMode.value === 'qrcode' &&
  (fieldKey === 'appID' || fieldKey === 'appSecret');
</script>
