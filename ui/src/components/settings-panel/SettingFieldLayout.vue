<template>
  <div
    :class="[
      'setting-field-layout',
      `setting-field-layout-columns-${columns}`,
      { 'setting-field-layout-padded': padded },
    ]"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    columns?: 1 | 2;
    padded?: boolean;
  }>(),
  {
    columns: 1,
    padded: false,
  }
);
</script>

<style scoped>
.setting-field-layout {
  display: grid;
  min-width: 0;
}

.setting-field-layout-padded {
  gap: var(--sd-space-sm);
  padding: var(--sd-space-md);
}

.setting-field-layout :deep(.n-form-item) {
  min-width: 0;
}

/*
 * Intermediate rows keep Naive UI's reserved feedback height, so validation
 * does not move later rows. Only an empty feedback wrapper in the visual final
 * row collapses. Real feedback stays in flow and expands the panel downward.
 */
.setting-field-layout
  :deep(.n-form > .n-form-item:last-child > .n-form-item-feedback-wrapper:empty),
.setting-field-layout > :deep(.n-form-item:last-child > .n-form-item-feedback-wrapper:empty) {
  min-height: 0;
}

@media (min-width: 760.1px) {
  .setting-field-layout-columns-2 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: var(--sd-space-lg);
  }

  .setting-field-layout-columns-2 > :deep(.n-form-item:last-child:nth-child(odd)),
  .setting-field-layout-columns-2 > :deep(.setting-field-span-full) {
    grid-column: 1 / -1;
  }

  .setting-field-layout-columns-2
    > :deep(.n-form-item:nth-last-child(2):nth-child(odd) > .n-form-item-feedback-wrapper:empty) {
    min-height: 0;
  }
}

@media (max-width: 639.9px) {
  .setting-field-layout-padded {
    padding: var(--sd-space-sm);
  }
}
</style>
