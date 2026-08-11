<template>
  <div :class="['setting-row', `setting-row-${layout}`, { 'setting-row-highlighted': highlighted }]">
    <div class="setting-row-copy">
      <div class="setting-row-label">{{ label }}</div>
      <p v-if="description" class="setting-row-description">{{ description }}</p>
    </div>
    <div class="setting-row-control">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label: string;
  description?: string;
  layout?: 'inline' | 'auto' | 'stacked';
  highlighted?: boolean;
}>(), {
  layout: 'auto',
});
</script>

<style scoped>
.setting-row {
  display: grid;
  grid-template-columns: minmax(13rem, 18rem) minmax(13rem, 26rem);
  align-items: center;
  gap: var(--sd-space-lg);
  min-height: 2.75rem;
  padding: var(--sd-space-xs) var(--sd-space-md);
  border-radius: 6px;
  transition: background-color 0.16s ease;
}

.setting-row:hover {
  background: var(--sd-bg-hover);
  background: color-mix(in srgb, var(--sd-bg-hover), transparent 35%);
}

.setting-row-stacked {
  align-items: stretch;
  flex-direction: column;
  gap: 0.45rem;
  padding-block: 0.62rem;
}

.setting-row-highlighted {
  background: var(--sd-bg-selected);
}

.setting-row-copy {
  min-width: 0;
}

.setting-row-label {
  color: var(--sd-text-primary);
  font-size: 0.92rem;
  font-weight: 500;
  line-height: 1.45;
}

.setting-row-description {
  margin: 0.12rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
  line-height: 1.4;
}

.setting-row-control {
  display: flex;
  justify-content: flex-start;
  min-width: 0;
}

.setting-row-inline .setting-row-control {
  min-width: 0;
}

.setting-row-inline {
  grid-template-columns: auto max-content;
}

.setting-row-stacked .setting-row-copy,
.setting-row-stacked .setting-row-control {
  width: 100%;
  min-width: 0;
}

.setting-row-stacked .setting-row-control {
  flex-basis: auto;
}

@media (max-width: 959.9px) {
  .setting-row {
    gap: var(--sd-space-md);
    padding-inline: var(--sd-space-sm);
  }

  .setting-row-copy {
    min-width: 0;
  }

  .setting-row-control {
    min-width: 0;
  }
}

@media (max-width: 780px) {
  .setting-row-auto,
  .setting-row-stacked {
    display: flex;
    align-items: stretch;
    flex-direction: column;
    gap: 0.35rem;
  }

  .setting-row-auto .setting-row-copy,
  .setting-row-auto .setting-row-control,
  .setting-row-stacked .setting-row-copy,
  .setting-row-stacked .setting-row-control {
    width: 100%;
    min-width: 0;
  }

  .setting-row {
    min-height: 2.55rem;
    padding: var(--sd-space-xs) var(--sd-space-sm);
  }

  .setting-row-label {
    font-size: 0.9rem;
  }

  .setting-row-description {
    margin-top: 0.08rem;
  }
}
</style>
