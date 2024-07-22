<script setup lang="ts">
const mode = ref<string>('list')

</script>

<template>
  <el-tabs v-model="mode" :stretch="true">
    <el-tab-pane label="黑白名单" name="list">
      <ban-list />
    </el-tab-pane>
    <el-tab-pane label="拉黑设置" name="config">
      <ban-config />
    </el-tab-pane>
  </el-tabs>

</template>

<style scoped lang="css">

</style>
