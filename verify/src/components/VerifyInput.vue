<script setup lang="ts">
import type VerifyResult from '@/types'

const result = ref<VerifyResult>()
</script>

<template>
  <div class="break-all">
    <n-input type="textarea" :autosize="{ minRows: 3 }"
             placeholder="请输入生成的神秘海豹码"
             class="" />
  </div>

  <n-flex v-if="result" vertical align="center"
          class="mt-8 text-base">
    <n-text v-if="result.success" type="success">校验通过</n-text>
    <n-text v-else type="error">校验失败</n-text>

    <n-text>海豹版本 {{ result.version }}</n-text>

    <n-text>于 {{ result.timestamp }} 生成</n-text>

    <n-text>平台</n-text>

    <n-text>操作人</n-text>
  </n-flex>
</template>

<style scoped>

</style>