<template>
  <div
    ref="root"
    class="responsive-data-view"
    :aria-label="props.ariaLabel"
    :data-view-mode="mode"
    :role="props.ariaLabel ? 'region' : undefined"
  >
    <slot v-if="mode === 'table'" name="table" />
    <slot v-else name="compact" />
  </div>
</template>

<script setup lang="ts">
import { computed, useTemplateRef } from 'vue';
import { useElementSize } from '@vueuse/core';
import { resolveCompactContainerMode } from '@/features/responsive/layout';

const props = withDefaults(
  defineProps<{
    compactAt?: number;
    ariaLabel?: string;
  }>(),
  {
    compactAt: 760,
    ariaLabel: undefined,
  }
);

defineSlots<{
  table(): unknown;
  compact(): unknown;
}>();

const root = useTemplateRef<HTMLElement>('root');
const { width } = useElementSize(root);
const mode = computed(() => resolveCompactContainerMode(width.value, props.compactAt));
</script>

<style scoped>
.responsive-data-view {
  width: 100%;
  min-width: 0;
}
</style>
