import { nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';
import { useJsSelection } from './useJsSelection';

const item = (filename: string) => ({ filename });

describe('useJsSelection', () => {
  it('supports multi-select and select-all', () => {
    const items = ref([item('a.js'), item('b.js')]);
    const selection = useJsSelection(items);

    selection.toggleSelected('a.js', true);
    expect(selection.isSelected('a.js')).toBe(true);
    expect(selection.selectedCount.value).toBe(1);
    expect(selection.someSelected.value).toBe(true);

    selection.toggleSelected('b.js', true);
    expect(selection.selectedCount.value).toBe(2);
    expect(selection.allSelected.value).toBe(true);
    expect(selection.someSelected.value).toBe(false);

    selection.toggleSelected('a.js', false);
    expect(selection.selectedCount.value).toBe(1);

    selection.toggleSelectAll(true);
    expect(selection.selectedCount.value).toBe(2);

    selection.toggleSelectAll(false);
    expect(selection.selectedCount.value).toBe(0);
  });

  it('drops selections that are no longer in the list', async () => {
    const items = ref([item('a.js'), item('b.js')]);
    const selection = useJsSelection(items);

    selection.toggleSelectAll(true);
    items.value = [item('b.js'), item('c.js')];
    await nextTick();

    expect(selection.isSelected('a.js')).toBe(false);
    expect(selection.selectedCount.value).toBe(1);
    expect(selection.isSelected('b.js')).toBe(true);
  });
});
