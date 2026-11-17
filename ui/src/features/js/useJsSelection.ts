import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue';

type SelectableItem = { filename: string };

/**
 * 列表勾选状态独立于查询数据，避免重新请求、排序或编辑后勾选被重置。
 * 选择结果以 filename 为键，翻页或列表刷新时自动剔除已不存在的条目。
 */
export function useJsSelection(items: MaybeRefOrGetter<readonly SelectableItem[]>) {
  const selectedFilenames = ref<Set<string>>(new Set());

  const isSelected = (filename: string) => selectedFilenames.value.has(filename);

  const selectedCount = computed(
    () => toValue(items).filter(item => selectedFilenames.value.has(item.filename)).length
  );
  const allSelected = computed(() => {
    const list = toValue(items);
    return list.length > 0 && selectedCount.value === list.length;
  });
  const someSelected = computed(() => selectedCount.value > 0 && !allSelected.value);

  function toggleSelected(filename: string, checked: boolean) {
    const next = new Set(selectedFilenames.value);
    if (checked) next.add(filename);
    else next.delete(filename);
    selectedFilenames.value = next;
  }

  function toggleSelectAll(checked: boolean) {
    const next = new Set(selectedFilenames.value);
    for (const item of toValue(items)) {
      if (checked) next.add(item.filename);
      else next.delete(item.filename);
    }
    selectedFilenames.value = next;
  }

  function clearSelected() {
    selectedFilenames.value = new Set();
  }

  watch(
    () =>
      toValue(items)
        .map(item => item.filename)
        .join('\n'),
    () => {
      if (selectedFilenames.value.size === 0) return;
      const filenames = new Set(toValue(items).map(item => item.filename));
      const next = new Set(
        [...selectedFilenames.value].filter(filename => filenames.has(filename))
      );
      if (next.size !== selectedFilenames.value.size) selectedFilenames.value = next;
    }
  );

  return {
    selectedFilenames,
    isSelected,
    selectedCount,
    allSelected,
    someSelected,
    toggleSelected,
    toggleSelectAll,
    clearSelected,
  };
}
