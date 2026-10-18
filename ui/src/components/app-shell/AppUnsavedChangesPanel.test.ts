// @vitest-environment happy-dom
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { ref, nextTick } from 'vue';
import { afterEach, expect, it, vi } from 'vitest';
import { useUnsavedChangesStore } from '@/features/unsavedChanges/store';
import AppUnsavedChangesPanel from './AppUnsavedChangesPanel.vue';

let wrapper: VueWrapper | undefined;
afterEach(() => wrapper?.unmount());

it('保存与放弃按钮执行对应来源操作，保存期间禁用，草稿恢复后隐藏', async () => {
  const pinia = createPinia();
  const store = useUnsavedChangesStore(pinia);
  const dirty = ref(true);
  const saving = ref(false);
  const save = vi.fn(() => {
    dirty.value = false;
  });
  const discard = vi.fn(() => {
    dirty.value = false;
  });
  store.registerSource('settings', { label: '设置', dirty, saving, save, discard });
  wrapper = mount(AppUnsavedChangesPanel, { global: { plugins: [pinia] } });

  const button = (label: string) => wrapper!.findAll('button').find(item => item.text() === label)!;
  expect(wrapper.text()).toContain('设置 未保存');
  saving.value = true;
  await nextTick();
  expect(button('保存设置').attributes('disabled')).toBeDefined();
  expect(button('放弃改动').attributes('disabled')).toBeDefined();
  saving.value = false;
  await nextTick();
  await button('保存设置').trigger('click');
  await flushPromises();
  expect(save).toHaveBeenCalledOnce();
  expect(discard).not.toHaveBeenCalled();
  expect(wrapper.find('[role="status"]').exists()).toBe(false);

  dirty.value = true;
  await nextTick();
  await button('放弃改动').trigger('click');
  await flushPromises();
  expect(discard).toHaveBeenCalledOnce();
  expect(wrapper.find('[role="status"]').exists()).toBe(false);
});

it('已保存但待重载时展示重载操作，没有撤销能力时不提供放弃按钮', async () => {
  const pinia = createPinia();
  const store = useUnsavedChangesStore(pinia);
  const reload = vi.fn();
  store.registerSource('reload', { kind: 'reload', label: '扩展', dirty: true, save: reload });
  wrapper = mount(AppUnsavedChangesPanel, { global: { plugins: [pinia] } });
  expect(wrapper.text()).toContain('扩展 待重载');
  expect(store.hasUnsavedChanges).toBe(false);
  const buttons = wrapper.findAll('button');
  expect(buttons.map(button => button.text())).toEqual(['立即重载']);
  await buttons[0]!.trigger('click');
  expect(reload).toHaveBeenCalledOnce();
});
