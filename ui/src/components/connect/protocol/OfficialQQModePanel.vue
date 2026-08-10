<template>
  <n-alert type="info" :show-icon="false" class="mb-4">
    <n-space vertical size="small">
      <n-radio-group v-model:value="mode">
        <n-radio-button value="manual">手动填写 AppID</n-radio-button>
        <n-radio-button value="qrcode">扫码登录</n-radio-button>
      </n-radio-group>
      <n-text depth="3">
        {{
          mode === 'qrcode'
            ? '扫码模式会在创建后生成二维码；AppID 与密钥留空即可。'
            : '手动模式会在添加前先测试机器人凭据，确认账号是否已存在。'
        }}
      </n-text>
    </n-space>
  </n-alert>
</template>

<script setup lang="ts">
import type { OfficialQQMode } from '@/features/connect/officialQQ';

const mode = defineModel<OfficialQQMode>({ required: true });
</script>
