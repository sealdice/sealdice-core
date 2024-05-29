<script setup lang="ts">
import { darkTheme, lightTheme } from 'naive-ui'

const isDark = useDark()
</script>

<template>
  <n-config-provider :theme="isDark ? darkTheme : lightTheme">
    <main-page/>
  </n-config-provider>
</template>

<style scoped>
</style>
