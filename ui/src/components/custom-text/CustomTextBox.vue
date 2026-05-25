<template>
  <n-collapse-item :title="group === '__others__' ? '无分组' : group" :name="group">
    <n-grid :cols="1" :x-gap="20">
      <n-grid-item>
        <slot name="values" />
      </n-grid-item>
    </n-grid>
  </n-collapse-item>
</template>

<script setup lang="ts">
defineProps<{
  group: string | '__others__';
}>();
</script>
