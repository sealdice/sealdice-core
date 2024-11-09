<script setup lang="ts">
import VerigyControl from '@/components/VerifyControl.vue'
</script>

<template>
  <n-layout class="h-screen light:bg-slate-100">
    <n-layout-header class="mx-auto mb-4 mt-8 max-w-screen-lg bg-transparent" content-class="px-8">
      <verify-header />
    </n-layout-header>

    <n-layout-content class="mx-auto max-w-screen-lg light:bg-slate-100" content-class="px-8">
      <verigy-control class="my-4" />
      <verify-input />
    </n-layout-content>
  </n-layout>
  <main></main>
</template>

<style scoped></style>
