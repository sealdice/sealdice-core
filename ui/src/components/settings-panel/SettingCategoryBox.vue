<template>
  <section :class="['setting-category-box', { 'setting-category-box-wide': wide }]">
    <n-thing class="setting-category-thing">
      <template #header>
        <span class="setting-category-heading">
          <span :id="headingId" class="setting-category-title">{{ title }}</span>
          <slot name="title-extra" />
        </span>
      </template>
      <template v-if="collapsible" #header-extra>
        <n-button
          text
          size="small"
          class="setting-category-toggle"
          :aria-label="expanded ? `收起${title}` : `展开${title}`"
          :aria-expanded="expanded"
          :aria-controls="panelId"
          @click="emit('toggle')"
        >
          {{ expanded ? '收起' : '展开' }}
        </n-button>
      </template>
    </n-thing>

    <div
      v-if="showPanel && (!collapsible || expanded)"
      :id="panelId"
      class="setting-category-panel"
      role="region"
      :aria-labelledby="headingId"
    >
      <slot name="notes" />
      <div
        :class="[
          'setting-field-layout',
          `setting-field-layout-columns-${columns}`,
          { 'setting-field-layout-padded': padded },
        ]"
      >
        <slot />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useId } from 'vue';

const panelId = `setting-category-panel-${useId()}`;
const headingId = `setting-category-heading-${useId()}`;

withDefaults(
  defineProps<{
    title: string;
    collapsible?: boolean;
    expanded?: boolean;
    wide?: boolean;
    padded?: boolean;
    columns?: 1 | 2;
    showPanel?: boolean;
  }>(),
  {
    showPanel: true,
    columns: 1,
    padded: false,
  }
);

const emit = defineEmits<{
  toggle: [];
}>();
</script>

<style scoped>
.setting-category-box {
  padding: var(--sd-space-xs) 0 var(--sd-space-sm);
}

.setting-category-box-wide {
  grid-column: 1 / -1;
}

.setting-category-thing {
  margin: 0 var(--sd-space-md) var(--sd-space-xs);
}

.setting-category-title {
  color: var(--sd-text-primary);
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.35;
}

.setting-category-heading {
  display: inline-flex;
  align-items: center;
  gap: var(--sd-space-xs);
  min-width: 0;
}

.setting-category-toggle {
  flex: 0 0 auto;
}

.setting-category-panel {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  overflow: hidden;
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

@media (max-width: 639.9px) {
  .setting-category-box {
    padding-bottom: var(--sd-space-xs);
  }

  .setting-category-thing {
    margin: 0 var(--sd-space-xs) var(--sd-space-2xs);
  }
}

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
