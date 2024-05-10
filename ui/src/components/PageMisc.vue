<template>
  <page-misc-settings v-if="category === 'base'" />
  <page-misc-backup v-if="category === 'backup'" />
  <page-misc-group v-if="category === 'group'" />
  <page-misc-ban-list v-if="category === 'ban'" />
  <page-misc-advanced-settings v-if="category === 'advanced'" @update:advanced-settings-show="(show) => emit('update:advanced-settings-show', show)"/>
</template>

<script lang="ts" setup>
import PageMiscSettings from "./misc/PageMiscSettings.vue"
import PageMiscBackup from "./misc/PageMiscBackup.vue"
import PageMiscGroup from "./misc/PageMiscGroup.vue"
import PageMiscBanList from "./misc/PageMiscBanList.vue"
import PageMiscAdvancedSettings from "~/components/misc/PageMiscAdvancedSettings.vue";

const props = defineProps<{ category: string }>();

const emit = defineEmits(['update:advanced-settings-show']);
</script>
