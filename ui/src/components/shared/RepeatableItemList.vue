<template>
  <section class="sd-repeatable-list">
    <ul v-if="renderedItems > 0" class="sd-repeatable-list__items">
      <li v-for="index in renderedItems" :key="index" class="sd-repeatable-list__item">
        <div class="sd-repeatable-list__fields">
          <slot name="item" :index="index - 1" :count="count" />
        </div>
        <n-button
          class="sd-repeatable-list__remove"
          type="error"
          size="small"
          secondary
          :aria-label="removeLabel"
          @click="emit('remove', index - 1)"
        >
          <template #icon>
            <n-icon><i-tabler-trash /></n-icon>
          </template>
          {{ removeLabel }}
        </n-button>
      </li>
    </ul>

    <div v-else class="sd-repeatable-list__empty">
      <slot name="empty">{{ emptyText }}</slot>
    </div>

    <div v-if="collapsible && collapsed && hiddenItems > 0" class="sd-repeatable-list__toggle">
      <n-button text size="small" @click="collapsed = false">
        {{ expandLabel }}（{{ hiddenItems }} 项）
      </n-button>
    </div>
    <div
      v-else-if="collapsible && !collapsed && hiddenItems > 0"
      class="sd-repeatable-list__toggle"
    >
      <n-button text size="small" @click="collapsed = true">{{ collapseLabel }}</n-button>
    </div>

    <div class="sd-repeatable-list__add">
      <n-button
        type="primary"
        secondary
        :loading="addLoading"
        :disabled="addDisabled"
        @click="emit('add')"
      >
        <template #icon>
          <n-icon><i-tabler-plus /></n-icon>
        </template>
        {{ addLabel }}
      </n-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    count: number;
    addLabel: string;
    removeLabel?: string;
    emptyText?: string;
    addDisabled?: boolean;
    addLoading?: boolean;
    collapsible?: boolean;
    maxVisible?: number;
    expandLabel?: string;
    collapseLabel?: string;
  }>(),
  {
    removeLabel: '删除',
    emptyText: '',
    addDisabled: false,
    addLoading: false,
    collapsible: false,
    maxVisible: 3,
    expandLabel: '展开其余',
    collapseLabel: '收起多余条目',
  }
);

const emit = defineEmits<{
  add: [];
  remove: [index: number];
}>();

const collapsed = ref(true);

watch(
  () => [props.count, props.collapsible],
  () => {
    // Reset collapse state only when the list structure meaningfully changes:
    // when it shrinks below the visible cap nothing needs expanding.
    if (!props.collapsible || props.count <= props.maxVisible) {
      collapsed.value = true;
    }
  }
);

const hiddenItems = computed(() => Math.max(0, props.count - props.maxVisible));
const renderedItems = computed(() => {
  if (!props.collapsible) return props.count;
  return collapsed.value ? Math.min(props.count, props.maxVisible) : props.count;
});
</script>

<style scoped>
.sd-repeatable-list {
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

.sd-repeatable-list__items {
  display: flex;
  margin: 0;
  padding: 0;
  flex-direction: column;
  list-style: none;
}

.sd-repeatable-list__item {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-md);
  padding: var(--sd-space-sm) var(--sd-space-md);
}

.sd-repeatable-list__item + .sd-repeatable-list__item {
  border-top: 1px solid var(--sd-border-soft);
}

.sd-repeatable-list__fields {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  align-items: center;
  gap: var(--sd-space-sm);
}

.sd-repeatable-list__remove {
  flex: 0 0 auto;
}

.sd-repeatable-list__empty {
  padding: var(--sd-space-md);
  color: var(--sd-text-muted);
}

.sd-repeatable-list__toggle {
  display: flex;
  justify-content: flex-start;
  padding: var(--sd-space-2xs) var(--sd-space-md);
  border-top: 1px solid var(--sd-border-soft);
}

.sd-repeatable-list__add {
  display: flex;
  justify-content: flex-start;
  padding: var(--sd-space-sm) var(--sd-space-md);
  border-top: 1px solid var(--sd-border-soft);
}

@media (max-width: 639.9px) {
  .sd-repeatable-list__item {
    padding: var(--sd-space-xs) var(--sd-space-sm);
  }

  .sd-repeatable-list__add {
    padding: var(--sd-space-xs) var(--sd-space-sm);
  }
}
</style>
