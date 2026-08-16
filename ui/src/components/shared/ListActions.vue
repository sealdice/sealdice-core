<template>
  <section v-if="$slots.default || $slots.end" class="sd-list-actions">
    <div v-if="$slots.default" class="sd-list-actions__start">
      <slot />
    </div>
    <div v-if="$slots.end" class="sd-list-actions__end">
      <slot name="end" />
    </div>
  </section>
</template>

<script setup lang="ts"></script>

<style scoped>
.sd-list-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--sd-space-sm) var(--sd-space-md);
}

.sd-list-actions__start,
.sd-list-actions__end {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sd-space-xs);
}

.sd-list-actions__start {
  flex: 1 1 auto;
}

.sd-list-actions__end {
  margin-left: auto;
  justify-content: flex-end;
}

.sd-list-actions :deep(.n-upload) {
  width: auto;
}
</style>
