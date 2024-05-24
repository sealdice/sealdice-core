<template>
  <page-test v-if="category === 'test'"/>
  <page-upload v-else-if="category === 'upload'"/>
</template>

<script setup lang="ts">
import PageTest from "~/components/tool/PageTest.vue";
import PageUpload from "~/components/tool/PageResource.vue";

const props = defineProps<{ category: string }>();
</script>
