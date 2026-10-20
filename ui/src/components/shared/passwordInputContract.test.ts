// @vitest-environment happy-dom
import { mount } from '@vue/test-utils';
import { expect, it } from 'vitest';
import type { FormConfigItem } from '@/api';
import DynamicForm from './DynamicForm.vue';
import { NInputNumber, NSwitch, NSelect } from 'naive-ui';

it('敏感字段默认隐藏，点击显隐后仍能提交输入的真实值', async () => {
  const schema: FormConfigItem[] = [
    {
      name: 'Token',
      field_name: 'token',
      input_type: 0,
      is_required: 1,
      default: '',
      placeholder: '',
      check_type: 0,
      sensitive: true,
      sub_option: null,
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

// 没有被专用插槽接管的字段仍需保留普通控件的约束。
it('动态表单保留敏感输入、整数边界和只读状态', async () => {
  const field = (field_name: string, input_type: number): FormConfigItem => ({
    name: field_name,
    field_name,
    input_type,
    is_required: 0,
    default: '',
    placeholder: '',
    check_type: 0,
    sub_option: null,
  });
  const wrapper = mount(DynamicForm, {
    props: {
      schema: [
        { ...field('token', 0), sensitive: true, readonly: true },
        { ...field('port', 1), minimum: 1, maximum: 65535 },
        field('enabled', 10),
        { ...field('mode', 12), sub_option: [{ label: 'QQ', value: 'QQ' }] },
      ],
      modelValue: { token: 'secret', port: 5500, enabled: false, mode: 'QQ' },
    },
  });
  try {
    expect(wrapper.get('input[type=password]').attributes('disabled')).toBeDefined();
    const number = wrapper.getComponent(NInputNumber);
    expect(number.props()).toMatchObject({ min: 1, max: 65535, precision: 0 });
    number.vm.$emit('update:value', 1234);
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({ port: 1234 });
    expect(wrapper.getComponent(NSwitch).props('value')).toBe(false);
    expect(wrapper.getComponent(NSelect).props('options')).toEqual([{ label: 'QQ', value: 'QQ' }]);
  } finally {
    wrapper.unmount();
  }
});
