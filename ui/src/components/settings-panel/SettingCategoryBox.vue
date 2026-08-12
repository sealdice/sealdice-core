<template>
  <section :class="['setting-category-box', { 'setting-category-box-wide': wide }]">
    <n-thing class="setting-category-thing">
      <template #header>
        <span :id="headingId" class="setting-category-title">{{ title }}</span>
      </template>
      <template v-if="description" #description>
        <span class="setting-category-description">{{ description }}</span>
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
      v-if="!collapsible || expanded"
      :id="panelId"
      class="setting-category-panel"
      role="region"
      :aria-labelledby="headingId"
    >
      <slot name="notes" />
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useId } from 'vue';

const panelId = `setting-category-panel-${useId()}`;
const headingId = `setting-category-heading-${useId()}`;

defineProps<{
  title: string;
  description?: string;
  collapsible?: boolean;
  expanded?: boolean;
  wide?: boolean;
}>();

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

.setting-category-description {
  color: var(--sd-text-muted);
  font-size: 0.82rem;
  line-height: 1.45;
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
</style>
