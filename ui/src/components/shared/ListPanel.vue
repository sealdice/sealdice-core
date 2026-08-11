<template>
  <section class="sd-list-panel">
    <div v-if="$slots.toolbar" class="sd-list-panel__toolbar">
      <slot name="toolbar" />
    </div>
    <div class="sd-list-panel__body">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts"></script>

<style scoped>
.sd-list-panel {
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-xs);
  background: var(--sd-bg-elevated);
}

.sd-list-panel__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  border-bottom: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated-soft);
}

.sd-list-panel__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 0.5rem;
  padding: 0.75rem;
}

@media (max-width: 640px) {
  .sd-list-panel__toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
