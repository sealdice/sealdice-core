<template>
  <n-space vertical size="small">
    <n-radio-group :value="webhookEnabled" :disabled="disabled" @update:value="updateValue">
      <n-radio :value="false">WebSocket（默认）</n-radio>
      <n-radio :value="true">Webhook</n-radio>
    </n-radio-group>
    <n-text depth="3">
      {{
        webhookEnabled
          ? 'Webhook：需要公网 IP，并在腾讯开放平台配置可访问的回调地址。'
          : 'WebSocket：无需公网回调地址，适合普通部署者。'
      }}
    </n-text>
  </n-space>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: unknown;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  'update:value': [value: boolean];
}>();

const webhookEnabled = computed(() => props.value === true || props.value === 'true');

function updateValue(value: string | number | boolean | null) {
  emit('update:value', value === true);
}
</script>
