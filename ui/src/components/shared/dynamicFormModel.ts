import type { FormConfigItem } from '@/api';

// 后端连接协议用 FormConfigItem 描述动态表单。
// 这个文件只处理“schema <-> 前端 model/payload”的纯转换，便于单元测试，
// 组件本身只负责渲染控件和 v-model。
export type DynamicFormModel = Record<string, unknown>;

export type DynamicFormValidation = {
  valid: boolean;
  missingFields: string[];
};

export const fieldKeyOf = (item: FormConfigItem): string => item.field_name;

// 构造初始值时尽量贴近后端 input_type 语义，避免 Naive UI 控件收到 undefined。
export const buildDynamicFormInitialModel = (schema: FormConfigItem[]): DynamicFormModel => {
  return schema.reduce<DynamicFormModel>((model, item) => {
    model[fieldKeyOf(item)] = defaultValueOf(item);
    return model;
  }, {});
};

export const validateDynamicFormModel = (
  schema: FormConfigItem[],
  model: DynamicFormModel
): DynamicFormValidation => {
  const missingFields = schema
    .filter(item => item.is_required === 1)
    .filter(item => isEmptyValue(model[fieldKeyOf(item)]))
    .map(fieldKeyOf);
  return {
    valid: missingFields.length === 0,
    missingFields,
  };
};

const defaultValueOf = (item: FormConfigItem): unknown => {
  if (item.default === '') {
    switch (item.input_type) {
      case 1:
        return null;
      case 10:
        return false;
      default:
        return '';
    }
  }
  switch (item.input_type) {
    case 1: {
      const value = Number(item.default);
      return Number.isFinite(value) ? value : null;
    }
    case 10:
      return item.default === 'true' || item.default === '1';
    default:
      return item.default;
  }
};

const isEmptyValue = (value: unknown): boolean => {
  if (value === undefined || value === null || value === '') return true;
  return false;
};
