<template>
  <TipBox type="info" class="mb-4">
    <n-space vertical size="small">
      <n-radio-group v-model:value="mode">
        <n-radio-button value="manual">手动填写机器人 ID</n-radio-button>
        <n-radio-button value="qrcode">扫码登录</n-radio-button>
      </n-radio-group>
      <n-text depth="3">
        {{
          mode === 'qrcode'
            ? '扫码模式会在创建后生成二维码，自动获取机器人 ID 和密钥。'
            : '手动模式需要填写机器人 ID 和密钥，添加前会先测试凭据。'
        }}
      </n-text>
    </n-space>
  </TipBox>
</template>

<script setup lang="ts">
import type { OfficialQQMode } from '@/features/connect/officialQQ';
import TipBox from '@/components/shared/TipBox.vue';

const mode = defineModel<OfficialQQMode>({ required: true });
</script>
