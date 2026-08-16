<template>
  <section class="connect-account-grid" aria-label="已接入账号" :aria-busy="loading">
    <template v-if="loading && connections.length === 0">
      <article v-for="index in 4" :key="index" class="connect-account-grid__skeleton">
        <n-card size="small" :bordered="true">
          <n-skeleton text :repeat="2" />
          <n-skeleton text style="margin-top: 1rem" />
          <n-skeleton text :repeat="3" style="margin-top: 1rem" />
        </n-card>
      </article>
    </template>

    <ConnectAccountCard
      v-for="endpoint in connections"
      :key="endpoint.id"
      :endpoint="endpoint"
      :workflow="workflows[endpoint.id] ?? null"
      :qr-code="qrCodes[endpoint.id]"
      :is-test-mode="isTestMode"
      :pending-action="pendingAction?.endpointId === endpoint.id ? pendingAction.action : null"
      :has-pending-operation="Boolean(pendingAction)"
      :action-error="operationErrors[endpoint.id] ?? ''"
      @show-q-r-code="emit('showQRCode', $event)"
      @edit="emit('edit', $event)"
      @toggle-enable="(endpoint, enable) => emit('toggleEnable', endpoint, enable)"
      @delete="emit('delete', $event)"
    />
  </section>
</template>

<script setup lang="ts">
import type { EndPointInfo, WorkflowResp } from '@/api';
import ConnectAccountCard from './ConnectAccountCard.vue';

defineProps<{
  connections: EndPointInfo[];
  workflows: Record<string, WorkflowResp>;
  qrCodes: Record<string, string>;
  loading: boolean;
  isTestMode: boolean;
  pendingAction: { endpointId: string; action: 'enable' | 'delete' } | null;
  operationErrors: Record<string, string>;
}>();

const emit = defineEmits<{
  showQRCode: [endpoint: EndPointInfo];
  edit: [endpoint: EndPointInfo];
  toggleEnable: [endpoint: EndPointInfo, enable: boolean];
  delete: [endpoint: EndPointInfo];
}>();
</script>

<style scoped>
.connect-account-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 24rem), 1fr));
  gap: var(--sd-space-md);
  min-width: 0;
}

.connect-account-grid__skeleton :deep(.n-card) {
  min-height: 17rem;
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}
</style>
