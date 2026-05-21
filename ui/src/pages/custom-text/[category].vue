<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import CustomTextEditor from '@/components/custom-text/CustomTextEditor.vue';

const props = defineProps<{ category?: string }>();
const route = useRoute();
const category = computed(() => {
  const routeParams = route.params as Record<string, string | string[] | undefined>;
  const routeCategory = routeParams.category;
  const fallback = Array.isArray(routeCategory) ? routeCategory[0] : routeCategory;
  return props.category ?? String(fallback ?? '');
});
</script>

<template>
  <CustomTextEditor :category="category" />
</template>
