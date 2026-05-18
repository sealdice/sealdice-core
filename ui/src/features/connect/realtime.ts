import { computed, ref, watch } from 'vue';
import type { EndPointInfo, WorkflowResp } from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { subscribeRealtimeEvent, useRealtimeClient } from '@/features/realtime/client';
import {
  applyConnectionList,
  applyConnectionQRCode,
  applyConnectionUpdate,
  applyConnectionWorkflow,
} from './realtimeState';

type ConnectionListPayload = {
  items?: EndPointInfo[] | null;
};

type ConnectionUpdatedPayload = {
  item?: EndPointInfo | null;
};

type ConnectionWorkflowPayload = {
  endpointId: string;
  workflow?: WorkflowResp | null;
};

type ConnectionQRCodePayload = {
  endpointId: string;
  img?: string | null;
};

const connections = ref<EndPointInfo[]>([]);
const workflows = ref<Record<string, WorkflowResp>>({});
const qrCodes = ref<Record<string, string>>({});
const ready = ref(false);

let initialized = false;

// 连接管理页不主动轮询连接列表，而是消费全局实时事件：
// imconnection/list 提供全量快照，updated/workflow/qrcode 提供增量变化。
// 这样二维码登录、连接状态变化可以实时反映到页面上。
function ensureInitialized(): void {
  if (initialized) return;
  initialized = true;

  subscribeRealtimeEvent<ConnectionListPayload>('imconnection/list', (payload) => {
    const nextState = applyConnectionList(
      connections.value,
      workflows.value,
      qrCodes.value,
      payload?.items ?? null,
    );
    connections.value = nextState.connections;
    workflows.value = nextState.workflows;
    qrCodes.value = nextState.qrCodes;
    ready.value = true;
  });

  subscribeRealtimeEvent<ConnectionUpdatedPayload>('imconnection/updated', (payload) => {
    connections.value = applyConnectionUpdate(connections.value, payload?.item ?? null);
  });

  subscribeRealtimeEvent<ConnectionWorkflowPayload>('imconnection/workflow', (payload) => {
    if (!payload) return;
    workflows.value = applyConnectionWorkflow(
      workflows.value,
      payload.endpointId,
      payload.workflow ?? null,
    );
  });

  subscribeRealtimeEvent<ConnectionQRCodePayload>('imconnection/qrcode', (payload) => {
    if (!payload) return;
    qrCodes.value = applyConnectionQRCode(
      qrCodes.value,
      payload.endpointId,
      payload.img ?? null,
    );
  });

  watch(
    hasAccessToken,
    (canAccess) => {
      if (!canAccess) {
        connections.value = [];
        workflows.value = {};
        qrCodes.value = {};
        ready.value = false;
      }
    },
    { immediate: true },
  );
}

export function useRealtimeConnections() {
  const realtime = useRealtimeClient();
  ensureInitialized();

  return {
    connections,
    workflows,
    qrCodes,
    ready,
    connected: realtime.connected,
    connecting: realtime.connecting,
    lastError: realtime.lastError,
    reconnect: realtime.reconnect,
    workflowOf: computed(() => (endpointId: string) => workflows.value[endpointId] ?? null),
    qrCodeOf: computed(() => (endpointId: string) => qrCodes.value[endpointId] ?? ''),
  };
}
