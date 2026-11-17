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
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--sd-space-sm);
}

.sd-page-header__copy {
  min-width: 0;
  flex: 1 1 auto;
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
    align-items: flex-start;
  }

  .sd-page-header__actions {
    justify-content: flex-start;
  }
}
</style>
