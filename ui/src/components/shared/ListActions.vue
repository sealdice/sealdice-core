<template>
  <section v-if="$slots.default" class="sd-list-actions">
    <slot />
  </section>
</template>

<script setup lang="ts"></script>

<style scoped>
.sd-list-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  justify-content: flex-start;
}
</style>
