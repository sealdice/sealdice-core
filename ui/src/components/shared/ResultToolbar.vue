<template>
  <div class="sd-result-toolbar">
    <div class="sd-result-toolbar__meta">
      <slot name="meta" />
    </div>
    <div v-if="$slots.default" class="sd-result-toolbar__actions">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.sd-result-toolbar {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 0;
  border-top: 1px solid var(--sd-border-soft);
  border-bottom: 1px solid var(--sd-border-soft);
}

.sd-result-toolbar__meta,
.sd-result-toolbar__actions {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.sd-result-toolbar__actions {
  justify-content: flex-end;
}

@media (max-width: 640px) {
  .sd-result-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .sd-result-toolbar__actions {
    justify-content: flex-start;
  }
}
</style>
