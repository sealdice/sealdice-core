<template>
  <section ref="toolbarRef" class="sd-query-toolbar">
    <ProSearchForm
      :form="form"
      :columns="columns"
      size="small"
      label-placement="left"
      :label-width="labelWidth"
      :cols="cols"
      :loading="loading"
      :search-button-props="{ type: 'primary', size: 'small' }"
      :reset-button-props="{ size: 'small' }"
      :collapse-button-props="false"
    />
  </section>
</template>

<script setup lang="ts" generic="T extends Record<string, any> = Record<string, unknown>">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';

const props = withDefaults(
  defineProps<{
    form: ReturnType<typeof createProSearchForm<T>>;
    columns: ProSearchFormColumns;
    loading?: boolean;
    labelWidth?: string | number;
    cols?: string | number;
  }>(),
  {
    loading: false,
    labelWidth: 84,
    cols: '1 s:2 l:3',
  }
);

const toolbarRef = ref<HTMLElement | null>(null);
let resizeObserver: ResizeObserver | undefined;
let mutationObserver: MutationObserver | undefined;

function markLastVisualRow() {
  const grid = toolbarRef.value?.querySelector<HTMLElement>('.n-grid');
  if (!grid) return;

  const items = Array.from(grid.children).filter(
    (item): item is HTMLElement =>
      item instanceof HTMLElement && item.getBoundingClientRect().height > 0
  );
  if (!items.length) return;

  const lastRowTop = Math.max(...items.map(item => item.getBoundingClientRect().top));
  items.forEach(item => {
    const isLastRow = Math.abs(item.getBoundingClientRect().top - lastRowTop) < 1;
    item.classList.toggle('sd-query-toolbar__last-row-item', isLastRow);
  });
}

async function refreshLastVisualRow() {
  await nextTick();
  markLastVisualRow();
}

onMounted(async () => {
  await refreshLastVisualRow();
  const root = toolbarRef.value;
  const grid = root?.querySelector<HTMLElement>('.n-grid');
  if (!root || !grid) return;

  resizeObserver = new ResizeObserver(markLastVisualRow);
  resizeObserver.observe(root);
  resizeObserver.observe(grid);
  mutationObserver = new MutationObserver(() => void refreshLastVisualRow());
  mutationObserver.observe(grid, { childList: true, subtree: true });
});

watch([() => props.cols, () => props.columns], () => void refreshLastVisualRow(), { deep: true });

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  mutationObserver?.disconnect();
});
</script>

<style scoped>
.sd-query-toolbar {
  padding: var(--sd-space-sm) var(--sd-space-md);
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated-soft);
}

/*
 * Intermediate rows retain Naive UI's reserved feedback height as their row
 * spacing. Only empty feedback in the current visual tail row collapses, so the
 * toolbar's top and bottom insets stay symmetric at every responsive column
 * count. Real feedback remains in flow.
 */
.sd-query-toolbar :deep(.sd-query-toolbar__last-row-item .n-form-item-feedback-wrapper:empty) {
  min-height: 0;
}
</style>
