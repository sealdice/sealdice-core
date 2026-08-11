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
import { useTemplateRef } from 'vue';
import { useResponsiveContainerMode } from '@/features/responsive/useResponsiveContainerMode';

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
const { mode } = useResponsiveContainerMode(root, { compactAt: () => props.compactAt });
</script>

<style scoped>
.responsive-data-view {
  width: 100%;
  min-width: 0;
}
</style>
