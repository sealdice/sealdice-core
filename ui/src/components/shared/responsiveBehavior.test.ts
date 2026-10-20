// @vitest-environment happy-dom
import { expect, it, vi } from 'vitest';
import { ref, nextTick } from 'vue';
import { mount } from '@vue/test-utils';
import ResponsiveDataView from './ResponsiveDataView.vue';

const width = ref(1000);
// Only the measurement boundary is controlled; component/hook reactivity is real.
vi.mock('@vueuse/core', () => ({ useElementSize: () => ({ width }) }));

it('renders exactly one presentation and responds to container size and threshold changes', async () => {
  const wrapper = mount(ResponsiveDataView, {
    props: { compactAt: 760, ariaLabel: 'Items' },
    slots: { table: '<div>Table content</div>', compact: '<div>Compact content</div>' },
  });
  try {
    expect(wrapper.text()).toBe('Table content');
    width.value = 500;
    await nextTick();
    expect(wrapper.text()).toBe('Compact content');
    await wrapper.setProps({ compactAt: 400 });
    expect(wrapper.text()).toBe('Table content');
    expect(wrapper.attributes('aria-label')).toBe('Items');
  } finally {
    wrapper.unmount();
  }
});
