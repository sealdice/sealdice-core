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

<script setup lang="ts"></script>

<style scoped>
.sd-result-toolbar {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
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
</style>
