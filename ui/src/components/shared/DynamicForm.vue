<script setup lang="ts">
import { computed } from 'vue';
import type { FormRules } from 'naive-ui';
import type { FormConfigItem } from '@/api';
import {
  buildDynamicFormPayload,
  fieldKeyOf,
  validateDynamicFormModel,
  type DynamicFormModel,
} from './dynamicFormModel';

type FieldSlotProps = {
  item: FormConfigItem;
  fieldKey: string;
  value: unknown;
  setValue: (value: unknown) => void;
};

const props = defineProps<{
  schema: FormConfigItem[];
  modelValue: DynamicFormModel;
  disabled?: boolean;
  labelWidth?: number | string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: DynamicFormModel];
  validChange: [valid: boolean];
}>();

defineSlots<{
  field?: (props: FieldSlotProps) => unknown;
}>();

const validation = computed(() => validateDynamicFormModel(props.schema, props.modelValue));

const rules = computed<FormRules>(() => {
  return props.schema.reduce<FormRules>((acc, item) => {
    const key = fieldKeyOf(item);
    if (item.is_required === 1) {
      acc[key] = {
        required: true,
        message: item.err_msg || `请填写${item.name}`,
        trigger: ['blur', 'change'],
      };
    }
    return acc;
  }, {});
});

const setValue = (key: string, value: unknown) => {
  const next = {
    ...props.modelValue,
    [key]: value,
  };
  emit('update:modelValue', next);
  emit('validChange', validateDynamicFormModel(props.schema, next).valid);
};

const optionList = (item: FormConfigItem) =>
  (item.sub_option ?? []).map(option => ({
    label: option.label,
    value: option.value,
  }));

const valueOf = (key: string) => props.modelValue[key];
const updateValue = (key: string) => (value: unknown) => setValue(key, value);

const getPayload = () => buildDynamicFormPayload(props.schema, props.modelValue);
const isValid = () => validation.value.valid;
const isDisabled = (item: FormConfigItem) => props.disabled || Boolean(item.readonly);

defineExpose({
  getPayload,
  isValid,
});
</script>

<template>
  <n-form
    class="dynamic-form"
    :model="modelValue"
    :rules="rules"
    :label-width="labelWidth ?? 108"
    label-placement="left"
  >
    <n-form-item
      v-for="item in schema"
      :key="fieldKeyOf(item)"
      :label="item.name"
      :path="fieldKeyOf(item)"
    >
      <slot
        name="field"
        :item="item"
        :field-key="fieldKeyOf(item)"
        :value="valueOf(fieldKeyOf(item))"
        :set-value="(value: unknown) => setValue(fieldKeyOf(item), value)"
      >
        <n-input-number
          v-if="item.input_type === 1"
          :value="(valueOf(fieldKeyOf(item)) as number | null)"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          @update:value="updateValue(fieldKeyOf(item))"
        />
        <n-switch
          v-else-if="item.input_type === 10"
          :value="Boolean(valueOf(fieldKeyOf(item)))"
          :disabled="isDisabled(item)"
          @update:value="updateValue(fieldKeyOf(item))"
        />
        <n-select
          v-else-if="item.input_type === 12"
          :value="(valueOf(fieldKeyOf(item)) as string | null)"
          :options="optionList(item)"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          @update:value="updateValue(fieldKeyOf(item))"
        />
        <n-radio-group
          v-else-if="item.input_type === 5"
          :value="(valueOf(fieldKeyOf(item)) as string | number | null)"
          :disabled="isDisabled(item)"
          @update:value="updateValue(fieldKeyOf(item))"
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
          :value="(valueOf(fieldKeyOf(item)) as Array<string | number>)"
          :disabled="isDisabled(item)"
          @update:value="updateValue(fieldKeyOf(item))"
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
        <n-date-picker
          v-else-if="item.input_type === 4 || item.input_type === 11"
          :value="(valueOf(fieldKeyOf(item)) as number | [number, number] | null)"
          :type="item.input_type === 11 ? 'datetimerange' : 'datetime'"
          :disabled="isDisabled(item)"
          @update:value="updateValue(fieldKeyOf(item))"
        />
        <n-input
          v-else
          :value="String(valueOf(fieldKeyOf(item)) ?? '')"
          :type="item.sensitive ? 'password' : 'text'"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          show-password-on="mousedown"
          @update:value="updateValue(fieldKeyOf(item))"
        />
      </slot>
      <template v-if="item.hint" #feedback>
        {{ item.hint }}
      </template>
    </n-form-item>
  </n-form>
</template>

<style scoped>
.dynamic-form {
  text-align: left;
}

.dynamic-form :deep(.n-input-number) {
  width: 100%;
}
</style>
