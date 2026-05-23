<script setup lang="ts">
import MagicSourcePanel from '@/components/magic/MagicSourcePanel.vue';
</script>

<template>
  <main class="magic-page">
    <MagicSourcePanel />
  </main>
</template>

<style scoped>
.magic-page {
  display: grid;
  gap: 16px;
  min-width: 0;
}
</style>
