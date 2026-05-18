import type { FormConfigItem } from '@/api';

export type DynamicFormModel = Record<string, unknown>;

export type DynamicFormValidation = {
  valid: boolean;
  missingFields: string[];
};

export const fieldKeyOf = (item: FormConfigItem): string => item.field_name || String(item.id);

export const buildDynamicFormInitialModel = (schema: FormConfigItem[]): DynamicFormModel => {
  return schema.reduce<DynamicFormModel>((model, item) => {
    model[fieldKeyOf(item)] = defaultValueOf(item);
    return model;
  }, {});
};

export const buildDynamicFormPayload = (
  schema: FormConfigItem[],
  model: DynamicFormModel
): Record<string, unknown> => {
  return schema.reduce<Record<string, unknown>>((payload, item) => {
    const key = fieldKeyOf(item);
    const value = model[key];
    if (value === undefined || value === null || value === '') {
      if (item.is_required === 1) payload[key] = value;
      return payload;
    }
    payload[key] = value;
    return payload;
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
  if (item.default_range) return item.default_range;
  if (item.default === '') {
    switch (item.input_type) {
      case 1:
        return null;
      case 6:
        return [];
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
    case 6:
      try {
        return JSON.parse(item.default) as unknown[];
      } catch {
        return [];
      }
    default:
      return item.default;
  }
};

const isEmptyValue = (value: unknown): boolean => {
  if (value === undefined || value === null || value === '') return true;
  if (Array.isArray(value)) return value.length === 0;
  return false;
};
