// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import { afterEach, expect, it, vi } from 'vitest';
import CustomTextCodeEditor from './CustomTextCodeEditor.vue';

const { create, destroy, setText } = vi.hoisted(() => ({
  create: vi.fn(),
  destroy: vi.fn(),
  setText: vi.fn(),
}));
vi.mock('@/features/customText/codeEditor', () => ({ createTextEditor: create }));
afterEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
  document.body.innerHTML = '';
});

function editorFixture() {
  create.mockImplementation(
    (
      host: HTMLElement,
      _text: string,
      selection: { anchor: number; head: number; scrollTop: number }
    ) => {
      const field = document.createElement('input');
      host.append(field);
      field.focus();
      return {
        snapshot: () => selection,
        setText,
        destroy: () => {
          field.remove();
          destroy();
        },
      };
    }
  );
}

it('100 idle fields create no full editors; focus and blur create and release just the active editor', async () => {
  vi.useFakeTimers();
  editorFixture();
  const fields = Array.from({ length: 100 }, () =>
    mount(CustomTextCodeEditor, { props: { modelValue: '{$t玩家}' }, attachTo: document.body })
  );
  try {
    expect(create).not.toHaveBeenCalled();
    const field = fields[0]!.get('textarea').element;
    field.focus();
    field.setSelectionRange(2, 4);
    await flushPromises();
    expect(create).toHaveBeenCalledTimes(1);
    expect(create.mock.calls[0]![2]).toMatchObject({ anchor: 2, head: 4 });
    fields[1]!.get('textarea').element.focus();
    await flushPromises();
    await vi.runOnlyPendingTimersAsync();
    expect(create).toHaveBeenCalledTimes(2);
    expect(destroy).toHaveBeenCalledTimes(1);
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
    await vi.runOnlyPendingTimersAsync();
    await nextTick();
    expect(destroy).toHaveBeenCalledTimes(2);
    expect(fields[0]!.get('textarea').isVisible()).toBe(true);
  } finally {
    fields.forEach(field => field.unmount());
  }
});

it('keeps IME composition in the native input and only activates once it has completed', async () => {
  editorFixture();
  const wrapper = mount(CustomTextCodeEditor, {
    props: { modelValue: '' },
    attachTo: document.body,
  });
  try {
    const field = wrapper.get('textarea');
    await field.trigger('compositionstart');
    field.element.focus();
    await flushPromises();
    expect(create).not.toHaveBeenCalled();
    await field.setValue('中文输入');
    await wrapper.setProps({ modelValue: '中文输入' });
    await field.trigger('compositionend');
    await flushPromises();
    expect(create).toHaveBeenCalledTimes(1);
    expect(create.mock.calls[0]![1]).toBe('中文输入');
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['中文输入']);
  } finally {
    wrapper.unmount();
  }
});

it('does not create an editor for a field that loses focus before the lazy import completes', async () => {
  editorFixture();
  const first = mount(CustomTextCodeEditor, {
    props: { modelValue: 'first' },
    attachTo: document.body,
  });
  const second = mount(CustomTextCodeEditor, {
    props: { modelValue: 'second' },
    attachTo: document.body,
  });
  try {
    first.get('textarea').element.focus();
    second.get('textarea').element.focus();
    await vi.dynamicImportSettled();
    await flushPromises();
    expect(first.get('textarea').isVisible()).toBe(true);
    expect(second.get('textarea').isVisible()).toBe(false);
    expect(second.element.contains(document.activeElement)).toBe(true);
  } finally {
    first.unmount();
    second.unmount();
  }
});

it('resizes by pointer or keyboard and retains the chosen height across focus changes', async () => {
  editorFixture();
  vi.useFakeTimers();
  const wrapper = mount(CustomTextCodeEditor, {
    props: { modelValue: '<script>literal</script>' },
    attachTo: document.body,
  });
  try {
    expect(wrapper.find('script').exists()).toBe(false);
    const handle = wrapper.get('[role="separator"]');
    const element = handle.element as HTMLElement;
    element.setPointerCapture = vi.fn();
    element.hasPointerCapture = () => true;
    element.releasePointerCapture = vi.fn();
    await handle.trigger('pointerdown', { pointerId: 1, button: 0, clientY: 100 });
    expect(element.setPointerCapture).toHaveBeenCalledWith(1);
    await handle.trigger('pointermove', { pointerId: 1, clientY: 230 });
    await handle.trigger('pointerup', { pointerId: 1 });
    expect(element.releasePointerCapture).toHaveBeenCalledWith(1);
    expect(handle.attributes('aria-valuenow')).toBe('226');
    wrapper.get('textarea').element.focus();
    await flushPromises();
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
    await vi.runOnlyPendingTimersAsync();
    expect(handle.attributes('aria-valuenow')).toBe('226');
    await handle.trigger('keydown', { key: 'ArrowDown' });
    expect(handle.attributes('aria-valuenow')).toBe('246');
  } finally {
    wrapper.unmount();
  }
});

it('reports editor creation failure, preserves editable text and selection, and retries on refocus', async () => {
  editorFixture();
  create.mockImplementationOnce((host: HTMLElement) => {
    const partialEditor = document.createElement('input');
    host.append(partialEditor);
    partialEditor.focus();
    throw new Error('Synthetic editor failure');
  });
  const wrapper = mount(CustomTextCodeEditor, {
    props: { modelValue: '{$tValue}' },
    attachTo: document.body,
  });
  try {
    const input = wrapper.get('textarea');
    input.element.setSelectionRange(2, 5);
    input.element.focus();
    await flushPromises();
    expect(wrapper.get('[role="status"]').text()).toContain('加载失败');
    expect(input.isVisible()).toBe(true);
    expect(input.element.value).toBe('{$tValue}');
    expect(document.activeElement).toBe(input.element);
    expect(input.element.selectionStart).toBe(2);
    expect(input.element.selectionEnd).toBe(5);
    expect(wrapper.findAll('.active-editor input')).toHaveLength(0);
    expect(wrapper.find('.passive-highlight .seal-variable').exists()).toBe(true);
    await input.setValue('示例输入');
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['示例输入']);
    await wrapper.setProps({ modelValue: '示例输入' });
    input.element.blur();
    input.element.focus();
    await flushPromises();
    expect(create).toHaveBeenCalledTimes(2);
    expect(create.mock.calls[1]![1]).toBe('示例输入');
    expect(wrapper.find('[role="status"]').exists()).toBe(false);
  } finally {
    wrapper.unmount();
  }
});
