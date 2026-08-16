<template>
  <AsyncFieldSection
    :loading="signInfoState.mode === 'loading'"
    :message="
      fieldKey === 'signServerName' && signInfoState.mode === 'manual-fallback'
        ? ''
        : signInfoState.message
    "
    :error="signInfoErrorMessage"
    @retry="emit('retry')"
  >
    <n-select
      v-if="fieldKey === 'signServerVersion'"
      :value="value as string"
      :options="signVersionOptions"
      :disabled="!signInfoState.canSelectVersion"
      placeholder="请选择签名版本"
      @update:value="emit('update:value', $event)"
    />
    <n-select
      v-else-if="!signInfoState.showCustomServerInput"
      :value="value as string"
      :options="signServers"
      :disabled="!signInfoState.canSelectServer"
      placeholder="请选择签名服务"
      @update:value="emit('update:value', $event)"
    />
    <n-input
      v-else
      :value="value as string"
      placeholder="请输入自定义签名地址"
      @update:value="emit('update:value', $event)"
    />
  </AsyncFieldSection>
</template>

<script setup lang="ts">
import type { SelectOption } from 'naive-ui';
import AsyncFieldSection from '@/components/shared/AsyncFieldSection.vue';
import type { SignInfoState } from '@/features/connect/signInfoState';

defineProps<{
  fieldKey: string;
  value: unknown;
  signInfoState: SignInfoState;
  signInfoErrorMessage: string;
  signVersionOptions: SelectOption[];
  signServers: SelectOption[];
}>();

const emit = defineEmits<{
  retry: [];
  'update:value': [value: unknown];
}>();
</script>
