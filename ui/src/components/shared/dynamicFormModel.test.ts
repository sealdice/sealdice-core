import type { FormConfigItem } from '@/api';
import {
  buildDynamicFormInitialModel,
  buildDynamicFormPayload,
  validateDynamicFormModel,
} from './dynamicFormModel';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const schema: FormConfigItem[] = [
  {
    id: 1,
    name: 'Token',
    field_name: 'token',
    input_type: 0,
    is_required: 1,
    default: '',
    placeholder: '',
    hint: '',
    err_msg: '',
    check_type: 0,
    sensitive: true,
    sub_option: null,
    size_range: { min: 0, max: 0 },
  },
  {
    id: 2,
    name: '端口',
    field_name: 'port',
    input_type: 1,
    is_required: 1,
    default: '5500',
    placeholder: '',
    hint: '',
    err_msg: '',
    check_type: 1,
    sub_option: null,
    size_range: { min: 0, max: 0 },
  },
  {
    id: 3,
    name: '启用',
    field_name: 'enabled',
    input_type: 10,
    is_required: 0,
    default: 'true',
    placeholder: '',
    hint: '',
    err_msg: '',
    check_type: 0,
    sub_option: null,
    size_range: { min: 0, max: 0 },
  },
];

const model = buildDynamicFormInitialModel(schema);
assertEqual(model.token, '');
assertEqual(model.port, 5500);
assertEqual(model.enabled, true);

const missing = validateDynamicFormModel(schema, model);
assertEqual(missing.valid, false);
assertDeepEqual(missing.missingFields, ['token']);

const payload = buildDynamicFormPayload(schema, { ...model, token: 'abc' });
assertDeepEqual(payload, {
  token: 'abc',
  port: 5500,
  enabled: true,
});
