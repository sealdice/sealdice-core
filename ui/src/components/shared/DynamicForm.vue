<template>
  <n-form
    class="dynamic-form"
    :model="modelValue"
    :rules="rules"
    :label-width="labelWidth ?? 108"
    :label-placement="labelPlacement ?? 'left'"
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
          :value="numberValueOf(item)"
          :min="item.minimum"
          :max="item.maximum"
          :precision="0"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          @update:value="(value: number | null) => setValue(fieldKeyOf(item), value)"
        />
        <n-switch
          v-else-if="item.input_type === 10"
          :value="Boolean(valueOf(fieldKeyOf(item)))"
          :disabled="isDisabled(item)"
          @update:value="(value: boolean) => setValue(fieldKeyOf(item), value)"
        />
        <n-select
          v-else-if="item.input_type === 12"
          :value="selectValueOf(item)"
          :options="optionList(item)"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          @update:value="(value: string | null) => setValue(fieldKeyOf(item), value)"
        />
        <n-input
          v-else
          :value="String(valueOf(fieldKeyOf(item)) ?? '')"
          :type="item.sensitive ? 'password' : 'text'"
          :disabled="isDisabled(item)"
          :placeholder="item.placeholder"
          show-password-on="click"
          @update:value="(value: string) => setValue(fieldKeyOf(item), value)"
        />
      </slot>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { FormProps, FormRules } from 'naive-ui';
import type { FormConfigItem } from '@/api';
import { fieldKeyOf, type DynamicFormModel } from './dynamicFormModel';

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
  labelPlacement?: FormProps['labelPlacement'];
  labelWidth?: number | string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: DynamicFormModel];
}>();

defineSlots<{
  field?: (props: FieldSlotProps) => unknown;
}>();

const rules = computed<FormRules>(() => {
  return props.schema.reduce<FormRules>((acc, item) => {
    const key = fieldKeyOf(item);
    if (item.is_required === 1) {
      acc[key] = {
        required: true,
        message: `请填写${item.name}`,
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
};

const optionList = (item: FormConfigItem) =>
  (item.sub_option ?? []).map(option => ({
    label: option.label,
    value: option.value,
  }));

const valueOf = (key: string) => props.modelValue[key];

// 模板里不能直接写 `x as A | B`：Prettier 会去掉外层括号，
// 之后 vue-eslint-parser 会把顶层 `|` 当成 Vue 2 过滤器语法而报错。
// 所以按控件类型在这里收口类型断言。
const numberValueOf = (item: FormConfigItem) => valueOf(fieldKeyOf(item)) as number | null;
const selectValueOf = (item: FormConfigItem) => valueOf(fieldKeyOf(item)) as string | null;
const isDisabled = (item: FormConfigItem) => props.disabled || Boolean(item.readonly);
</script>

<style scoped>
.dynamic-form {
  text-align: left;
}

.dynamic-form :deep(.n-input-number) {
  width: 100%;
}
</style>
