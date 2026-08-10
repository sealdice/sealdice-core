<template>
  <header class="sd-page-header">
    <div class="sd-page-header__copy">
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div v-if="$slots.default" class="sd-page-header__actions">
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
}>();
</script>

<style scoped>
.sd-page-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
}

.sd-page-header__copy {
  min-width: 0;
}

.sd-page-header h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.25;
}

.sd-page-header p {
  max-width: 60rem;
  margin: 0.4rem 0 0;
  color: var(--sd-text-secondary);
  line-height: 1.5;
}

.sd-page-header__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
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
