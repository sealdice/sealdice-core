<template>
  <page-misc-settings v-if="category === 'base'" />
  <page-misc-backup v-if="category === 'backup'" />
  <page-misc-group v-if="category === 'group'" />
  <page-misc-ban-list v-if="category === 'ban'" />
</template>

<script lang="ts" setup>
import PageMiscSettings from "./PageMiscSettings.vue"
import PageMiscBackup from "./PageMiscBackup.vue"
import PageMiscGroup from "./PageMiscGroup.vue"
import PageMiscBanList from "./PageMiscBanList.vue"
const props = defineProps<{ category: string }>();
</script>
