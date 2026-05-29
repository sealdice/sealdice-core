<template>
  <AppShell content-mode="wide" container-mode="workspace">
    <slot />
  </AppShell>
</template>

<script setup lang="ts">
import AppShell from '@/components/app-shell/AppShell.vue';
</script>
