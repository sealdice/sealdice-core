<template>
  <section class="sd-list-workspace">
    <slot />
  </section>
</template>

<script setup lang="ts"></script>

<style scoped>
.sd-list-workspace {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-space-md);
}
</style>
