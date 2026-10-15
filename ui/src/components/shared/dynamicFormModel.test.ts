import type { FormConfigItem } from '@/api';
import {
  buildDynamicFormInitialModel,
  buildDynamicFormPayload,
  validateDynamicFormModel,
} from './dynamicFormModel';
import { expect, it } from 'vitest';

it('动态表单解析默认值并验证必填字段', async () => {
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
  expect(model.token).toBe('');
  expect(model.port).toBe(5500);
  expect(model.enabled).toBe(true);

  const missing = validateDynamicFormModel(schema, model);
  expect(missing.valid).toBe(false);
  expect(missing.missingFields).toEqual(['token']);

  const payload = buildDynamicFormPayload(schema, { ...model, token: 'abc' });
  expect(payload).toEqual({
    token: 'abc',
    port: 5500,
    enabled: true,
  });
});
