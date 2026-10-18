<template>
  <div class="async-field-section">
    <TipBox v-if="signInfoErrorMessage" type="error" class="section-alert">
      <n-flex justify="space-between" align="center" :wrap="false">
        <span>{{ signInfoErrorMessage }}</span>
        <n-button size="small" tertiary :disabled="disabled" @click="emit('retry')">重试</n-button>
      </n-flex>
    </TipBox>
    <n-spin :show="signInfoState.mode === 'loading'">
      <div class="section-body">
        <n-text
          v-if="
            signInfoState.message &&
            !(fieldKey === 'signServerName' && signInfoState.mode === 'manual-fallback')
          "
          depth="3"
          class="section-message"
        >
          {{ signInfoState.message }}
        </n-text>
        <n-select
          v-if="fieldKey === 'signServerVersion'"
          :value="value as string"
          :options="signVersionOptions"
          :disabled="disabled || !signInfoState.canSelectVersion"
          placeholder="请选择签名版本"
          @update:value="emit('update:value', $event)"
        />
        <n-select
          v-else-if="!signInfoState.showCustomServerInput"
          :value="value as string"
          :options="signServers"
          :disabled="disabled || !signInfoState.canSelectServer"
          placeholder="请选择签名服务"
          @update:value="emit('update:value', $event)"
        />
        <n-input
          v-else
          :disabled="disabled"
          :value="value as string"
          placeholder="请输入自定义签名地址"
          @update:value="emit('update:value', $event)"
        />
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import type { SelectOption } from 'naive-ui';
import TipBox from '@/components/shared/TipBox.vue';
import type { SignInfoState } from '@/features/connect/signInfoState';

defineProps<{
  fieldKey: string;
  disabled: boolean;
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

<style scoped>
.async-field-section {
  width: 100%;
}

.section-alert {
  margin-bottom: 0.75rem;
}

.section-body {
  width: 100%;
}

.section-message {
  display: block;
  font-size: 0.82rem;
  line-height: 1.45;
  margin-bottom: 0.5rem;
}
</style>
