<template>
  <page-js v-if="category === 'js'" />
  <page-custom-reply v-if="category === 'reply'" />
  <page-misc-deck v-if="category === 'deck'" />
  <page-help-doc v-if="category === 'helpdoc'" />
</template>

<script lang="ts" setup>
import PageJs from "./mod/PageJs.vue"
import PageMiscDeck from "./mod/PageMiscDeck.vue"
import PageHelpDoc from "./mod/PageHelpDoc.vue";

const props = defineProps<{ category: string }>();
</script>
