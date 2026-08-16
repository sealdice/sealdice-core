<template>
  <div class="sd-section-flow">
    <OfficialQQModePanel v-if="protocolModule.formKind === 'officialqq'" v-model="officialQqMode" />

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
          show-password-on="click"
          @update:value="setValue"
        />
        <n-input-number
          v-else-if="item.input_type === 1"
          :value="asNumberValue(value)"
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
        <n-select
          v-else-if="item.input_type === 12"
          :value="asSelectValue(value)"
          :options="optionList(item)"
          :disabled="submitting || testModeDisabled"
          :placeholder="item.placeholder"
          @update:value="setValue"
        />
        <n-radio-group
          v-else-if="item.input_type === 5"
          :value="asRadioValue(value)"
          :disabled="submitting || testModeDisabled"
          @update:value="setValue"
        >
          <n-radio-button
            v-for="option in optionList(item)"
            :key="String(option.value)"
            :value="option.value"
          >
            {{ option.label }}
          </n-radio-button>
        </n-radio-group>
        <n-checkbox-group
          v-else-if="item.input_type === 6"
          :value="asCheckboxValue(value)"
          :disabled="submitting || testModeDisabled"
          @update:value="setValue"
        >
          <n-space>
            <n-checkbox
              v-for="option in optionList(item)"
              :key="String(option.value)"
              :value="option.value"
            >
              {{ option.label }}
            </n-checkbox>
          </n-space>
        </n-checkbox-group>
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
import { getConnectProtocolModule } from '@/features/connect/protocols';
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

const protocolModule = computed(() => getConnectProtocolModule(props.protocol?.key ?? ''));

// 模板里不能直接写 `value as number | null`：Prettier 会去掉外层括号，
// 之后 vue-eslint-parser 会把顶层 `|` 当成 Vue 2 过滤器语法而报错。
const asNumberValue = (value: unknown) => value as number | null;
const asSelectValue = (value: unknown) => value as string | null;
const asRadioValue = (value: unknown) => value as string | number | null;
const asCheckboxValue = (value: unknown) => value as Array<string | number>;

const optionList = (item: FormConfigItem): SelectOption[] =>
  (item.sub_option ?? []).map(option => ({
    label: option.label,
    value: option.value,
  }));

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
