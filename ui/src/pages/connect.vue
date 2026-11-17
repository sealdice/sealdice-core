<template>
  <main class="connect-page sd-page-flow">
    <PageHeader title="账号设置">
      <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
        <template #icon
          ><n-icon><i-tabler-plus /></n-icon
        ></template>
        添加账号
      </n-button>
    </PageHeader>

    <TipBox v-if="connectionsErrorText" type="error">
      <div class="connect-alert-content">
        <span>{{ connectionsErrorText }}</span>
        <n-button size="small" secondary @click="retryConnections">
          <template #icon
            ><n-icon><i-tabler-refresh /></n-icon
          ></template>
          重试
        </n-button>
      </div>
    </TipBox>

    <ListEmptyState v-if="connections.length === 0 && connectionsReady" description="暂无接入账号">
      <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
        添加账号
      </n-button>
    </ListEmptyState>

    <section
      v-else
      class="connect-account-grid"
      aria-label="已接入账号"
      :aria-busy="connectionsLoading"
    >
      <template v-if="connectionsLoading && connections.length === 0">
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
        :is-test-mode="isTestMode"
        :pending-action="
          pendingEndpointAction?.endpointId === endpoint.id ? pendingEndpointAction.action : null
        "
        :has-pending-operation="Boolean(pendingEndpointAction)"
        :action-error="endpointOperationErrors[endpoint.id] ?? ''"
        @show-q-r-code="openQRCode($event)"
        @edit="editingEndpoint = $event"
        @toggle-enable="confirmEnable"
        @delete="confirmDelete($event)"
      />
    </section>

    <ConnectCreateWizard v-model:show="dialogVisible" @created="onCreated" />

    <ConnectEditDialog :endpoint="editingEndpoint" @close="editingEndpoint = null" />

    <n-modal
      v-model:show="qrDialogVisible"
      preset="dialog"
      title="登录二维码"
      class="qrcode-dialog"
      :show-icon="false"
    >
      <n-space vertical align="center" size="large">
        <n-image v-if="activeQRCode" :src="activeQRCode" width="280" preview-disabled />
        <n-empty v-else description="当前没有可用二维码" />
        <n-button size="small" secondary @click="realtime.reconnect"> 刷新连接 </n-button>
      </n-space>
    </n-modal>
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { storeToRefs } from 'pinia';
import {
  getSdApiV2ImconnectionOptions,
  getSdApiV2ImconnectionQueryKey,
  deleteSdApiV2ImconnectionById,
  putSdApiV2ImconnectionByIdEnable,
  type EndPointInfo,
} from '@/api';
import ConnectCreateWizard from '@/components/connect/ConnectCreateWizard.vue';
import ConnectEditDialog from '@/components/connect/ConnectEditDialog.vue';
import ConnectAccountCard from '@/components/connect/ConnectAccountCard.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { getErrorMessage } from '@/features/auth/error';
import { getEndpointTargetLabel } from '@/features/connect/endpointDisplay';
import { useRealtimeConnectionsStore } from '@/features/connect/realtimeStore';
import { useRealtimeClientStore } from '@/features/realtime/store';
import { useTestMode } from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';
import { useAuthStore } from '@/features/auth/store';

const { hasAccessToken } = storeToRefs(useAuthStore());
const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const realtimeConnections = useRealtimeConnectionsStore();
realtimeConnections.ensureInitialized();
const realtime = useRealtimeClientStore();
const { connections } = storeToRefs(realtimeConnections);
const { isTestMode } = useTestMode();
const dialogVisible = ref(false);
const editingEndpoint = ref<EndPointInfo | null>(null);
const qrDialogVisible = ref(false);
const qrDialogEndpointId = ref('');
const endpointOperationErrors = ref<Record<string, string>>({});
const connectionsQuery = useQuery({ ...getSdApiV2ImconnectionOptions(), enabled: hasAccessToken });
const connectionsReady = computed(
  () =>
    realtimeConnections.ready || connectionsQuery.isSuccess.value || connectionsQuery.isError.value
);
const connectionsErrorText = computed(() =>
  connectionsQuery.isError.value && !realtimeConnections.ready
    ? getErrorMessage(connectionsQuery.error.value, '账号列表读取失败')
    : ''
);
const connectionsLoading = computed(() => hasAccessToken.value && !connectionsReady.value);
const qrEndpoint = computed(() =>
  connections.value.find(item => item.id === qrDialogEndpointId.value)
);
const activeQRCode = computed(() => qrEndpoint.value?.qrcode ?? '');
watch(
  () => qrEndpoint.value?.workflow?.state,
  state => {
    if (qrDialogVisible.value && state === 'success') {
      qrDialogVisible.value = false;
      qrDialogEndpointId.value = '';
    }
  }
);
watch(
  connectionsQuery.data,
  data => {
    if (data)
      realtimeConnections.applyInitialSnapshot(data.item.items ?? null, !realtime.connected);
  },
  { immediate: true }
);

type EndpointOperation =
  | { id: string; action: 'delete' }
  | { id: string; action: 'enable'; enable: boolean };
const operationMutation = useMutation({
  mutationFn: async (operation: EndpointOperation) => {
    if (operation.action === 'delete') {
      await deleteSdApiV2ImconnectionById({ path: { id: operation.id }, throwOnError: true });
    } else {
      await putSdApiV2ImconnectionByIdEnable({
        path: { id: operation.id },
        body: { enable: operation.enable },
        throwOnError: true,
      });
    }
  },
  onMutate: operation => {
    delete endpointOperationErrors.value[operation.id];
  },
  onSuccess: async (_, operation) => {
    message.success(operation.action === 'delete' ? '账号已删除' : '账号状态已更新');
    // SSE 断开时仍通过 HTTP 更新；在线时保留已经接收的实时状态。
    await queryClient.invalidateQueries({ queryKey: getSdApiV2ImconnectionQueryKey() });
  },
  onError: (error, operation) => {
    const fallback = operation.action === 'delete' ? '账号删除失败' : '账号状态更新失败';
    endpointOperationErrors.value[operation.id] = getErrorMessage(error, fallback);
    notifyError(error, message, fallback);
  },
});
const pendingEndpointAction = computed(() => {
  const operation = operationMutation.variables.value;
  return operationMutation.isPending.value && operation
    ? { endpointId: operation.id, action: operation.action }
    : null;
});
function runEndpointOperation(operation: EndpointOperation) {
  if (isTestMode.value || operationMutation.isPending.value) return;
  return operationMutation.mutateAsync(operation).catch(() => undefined);
}
function confirmDelete(endpoint: EndPointInfo) {
  if (isTestMode.value || operationMutation.isPending.value) return;
  dialog.warning({
    title: '删除账号',
    content: `确认删除账号「${getEndpointTargetLabel(endpoint)}」吗？删除账号不会影响人物卡和 logs 等数据。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => runEndpointOperation({ id: endpoint.id, action: 'delete' }),
  });
}
function confirmEnable(endpoint: EndPointInfo, enable: boolean) {
  if (isTestMode.value || operationMutation.isPending.value) return;
  dialog.warning({
    title: '修改账号状态',
    content: `确认${enable ? '启用' : '禁用'}账号「${getEndpointTargetLabel(endpoint)}」吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => runEndpointOperation({ id: endpoint.id, action: 'enable', enable }),
  });
}
function openCreateDialog() {
  if (!isTestMode.value) dialogVisible.value = true;
}
function openQRCode(endpoint: EndPointInfo) {
  qrDialogEndpointId.value = endpoint.id;
  qrDialogVisible.value = true;
}
function onCreated(endpoint: EndPointInfo, showQRCode: boolean) {
  if (showQRCode) openQRCode(endpoint);
}
function retryConnections() {
  void connectionsQuery.refetch();
}
</script>

<style scoped>
.connect-page {
  text-align: left;
}

.connect-alert-content {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--sd-space-sm);
}

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
