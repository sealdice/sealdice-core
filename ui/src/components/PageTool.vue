<template>
  <page-test v-if="category === 'test'"/>
  <page-resource v-else-if="category === 'resource'"/>
</template>

<script setup lang="ts">
const props = defineProps<{ category: string }>();
</script>
