<script setup lang="ts">
import { computed } from 'vue';
import { getSensitiveTag } from '@/features/censor/viewModel';

const props = defineProps<{
  level?: number | string;
  label?: string;
}>();

const tag = computed(() => getSensitiveTag(props.level ?? 0));
const text = computed(() => props.label || tag.value.label);
</script>

<template>
  <n-tag size="small" :type="tag.type" :bordered="false">
    {{ text }}
  </n-tag>
</template>
