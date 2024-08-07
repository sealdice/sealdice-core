<template>
  <template v-if="group !== '__others__'">
    <el-collapse-item :title="group" :name="group">
      <el-row :gutter="20">
        <slot name="values"/>
      </el-row>
    </el-collapse-item>
  </template>
  <template v-else>
    <el-collapse-item title="无分组" :name="group">
      <el-row :gutter="20">
        <slot name="values"/>
      </el-row>
    </el-collapse-item>
  </template>
</template>

<script setup lang="ts">
defineProps<{
  group: string | '__others__'
}>()
</script>

<style scoped lang="css">
.text-group-title {
  border-top: 1px solid var(--el-border-color);
}
</style>