<template>
  <page-misc-settings v-if="category === 'base'" />
  <page-misc-backup v-if="category === 'backup'" />
</template>

<script lang="ts" setup>
import PageMiscSettings from "./PageMiscSettings.vue"
import PageMiscBackup from "./PageMiscBackup.vue"
const props = defineProps<{ category: string }>();
</script>
