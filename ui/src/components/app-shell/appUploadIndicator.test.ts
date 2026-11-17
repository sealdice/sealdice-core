// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createPinia, disposePinia, setActivePinia } from 'pinia';
import { mount, type VueWrapper } from '@vue/test-utils';
import AppUploadIndicator from './AppUploadIndicator.vue';
import { useUploadStore } from '@/features/upload/store';

vi.mock('@/features/upload/fileHash', () => ({
  hashFile: async () => {
    throw new Error('read failed');
  },
}));
const pinia = createPinia();
let wrapper: VueWrapper;
afterEach(() => {
  wrapper?.unmount();
  disposePinia(pinia);
});

it('retries a failed task without creating a phantom upload, and cancel removes it', async () => {
  setActivePinia(pinia);
  const store = useUploadStore();
  await store.enqueue([new File(['x'], 'a.png')], 'resource');
  const Popover = defineComponent({
    setup(_, { slots }) {
      return () => h('div', [slots.trigger?.(), slots.default?.()]);
    },
  });
  wrapper = mount(AppUploadIndicator, {
    global: { plugins: [pinia], stubs: { NPopover: Popover, Popover } },
  });
  expect(wrapper.text(), wrapper.html()).toContain('a.png');
  const button = (label: string) => wrapper.findAll('button').find(item => item.text() === label)!;
  await button('重试').trigger('click');
  await vi.waitFor(() => expect(store.tasks[0]?.status).toBe('error'));
  expect(store.hasActive).toBe(false);
  expect(wrapper.text()).toContain('read failed');
  await button('取消').trigger('click');
  expect(store.tasks).toHaveLength(0);
  expect(wrapper.text()).toBe('');
});
