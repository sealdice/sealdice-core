<template>
  <header class="sd-page-header">
    <div class="sd-page-header__copy">
      <h1>{{ title }}</h1>
    </div>
    <div v-if="$slots.default" class="sd-page-header__actions">
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{ title: string }>();
</script>

<style scoped>
.sd-page-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sd-space-md);
  padding-bottom: var(--sd-space-md);
  border-bottom: 1px solid var(--sd-border-soft);
}

.sd-page-header__copy {
  min-width: 0;
}

.sd-page-header h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: var(--sd-page-title-size);
  font-weight: var(--sd-page-title-weight);
  line-height: var(--sd-page-title-line-height);
}

.sd-page-header__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: var(--sd-space-sm);
  flex-wrap: wrap;
}

@media (max-width: 640px) {
  .sd-page-header {
    grid-template-columns: 1fr;
    align-items: start;
  }

  .sd-page-header__actions {
    justify-content: flex-start;
  }
}
</style>
