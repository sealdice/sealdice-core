// @vitest-environment happy-dom
import { afterEach, expect, it } from 'vitest';
import { mount, type VueWrapper } from '@vue/test-utils';
import RepeatableList from './RepeatableList.vue';
import RepeatableItem from './RepeatableItem.vue';

let wrapper: VueWrapper;
afterEach(() => wrapper?.unmount());

it('emits add from the visible action and prevents it while disabled', async () => {
  wrapper = mount(RepeatableList, {
    props: { empty: true, emptyText: 'No entries', addLabel: 'Add entry' },
  });
  expect(wrapper.text()).toContain('No entries');
  const button = wrapper.get('button');
  await button.trigger('click');
  expect(wrapper.emitted('add')).toHaveLength(1);
  await wrapper.setProps({ addDisabled: true });
  expect((button.element as HTMLButtonElement).disabled).toBe(true);
  await button.trigger('click');
  expect(wrapper.emitted('add')).toHaveLength(1);
});

it('emits removal for the selected item and respects disabled state', async () => {
  wrapper = mount(RepeatableItem, { props: { title: 'Entry', removeLabel: 'Remove entry' } });
  const remove = wrapper.get('button[aria-label="Remove entry"]');
  await remove.trigger('click');
  expect(wrapper.emitted('remove')).toHaveLength(1);
  await wrapper.setProps({ disabled: true });
  await remove.trigger('click');
  expect(wrapper.emitted('remove')).toHaveLength(1);
});
