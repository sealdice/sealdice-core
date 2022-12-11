<template>
  <page-misc-settings v-if="category === 'base'" />
  <page-misc-backup v-if="category === 'backup'" />
  <page-misc-group v-if="category === 'group'" />
  <page-misc-ban-list v-if="category === 'ban'" />
</template>

<script lang="ts" setup>
import PageMiscSettings from "./misc/PageMiscSettings.vue"
import PageMiscBackup from "./misc/PageMiscBackup.vue"
import PageMiscGroup from "./misc/PageMiscGroup.vue"
import PageMiscBanList from "./misc/PageMiscBanList.vue"

const props = defineProps<{ category: string }>();
</script>
