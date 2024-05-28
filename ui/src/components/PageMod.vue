<template>
  <page-js v-if="category === 'js'" />
  <page-custom-reply v-if="category === 'reply'" />
  <page-misc-deck v-if="category === 'deck'" />
  <page-help-doc v-if="category === 'helpdoc'" />
  <page-story v-if="category === 'story'" />
  <page-censor v-if="category === 'censor'" />
</template>

<script lang="ts" setup>
const props = defineProps<{ category: string }>();
</script>
