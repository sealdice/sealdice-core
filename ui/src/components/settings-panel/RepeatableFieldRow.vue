<template>
  <div class="repeatable-field-row">
    <span class="repeatable-field-row__label">{{ label }}</span>
    <div class="repeatable-field-row__control">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ label: string }>();
</script>

<style scoped>
.repeatable-field-row {
  display: grid;
  min-width: 0;
  grid-template-columns: 4.5rem minmax(0, 1fr);
  align-items: center;
  gap: var(--sd-space-sm);
}

.repeatable-field-row__label {
  color: var(--sd-text-muted);
  font-size: 0.85rem;
  line-height: 1.4;
}

.repeatable-field-row__control {
  min-width: 0;
}

@media (max-width: 640px) {
  .repeatable-field-row {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--sd-space-xs);
  }
}
</style>
