// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { expect, it } from 'vitest';
import type { FormConfigItem } from '@/api';
import DynamicForm from './DynamicForm.vue';

it('敏感字段默认隐藏，点击显隐后仍能提交输入的真实值', async () => {
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
  ];
  const wrapper = mount(DynamicForm, { props: { schema, modelValue: { token: 'secret' } } });
  try {
    const input = wrapper.get('input');
    expect(input.attributes('type')).toBe('password');
    await wrapper.get('.n-input__eye').trigger('click');
    expect(input.attributes('type')).toBe('text');
    await input.setValue('updated-token');
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({ token: 'updated-token' });
    await wrapper.get('.n-input__eye').trigger('click');
    expect(input.attributes('type')).toBe('password');
  } finally {
    wrapper.unmount();
  }
});
