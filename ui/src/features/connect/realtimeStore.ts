import { defineStore } from 'pinia';
import { computed, ref, shallowRef, watch } from 'vue';
import type { EndPointInfo, WorkflowResp } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import { useRealtimeClientStore } from '@/features/realtime/store';
import {
  applyConnectionQRCode,
  applyConnectionSnapshot,
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

export const useRealtimeConnectionsStore = defineStore('connect-realtime', () => {
  const authStore = useAuthStore(appPinia);
  const realtimeStore = useRealtimeClientStore(appPinia);

  const connections = ref<EndPointInfo[]>([]);
  const workflows = ref<Record<string, WorkflowResp>>({});
  const qrCodes = ref<Record<string, string>>({});
  const ready = shallowRef(false);

  let initialized = false;

  function replaceSnapshot(nextConnections?: EndPointInfo[] | null): void {
    const nextState = applyConnectionSnapshot(
      connections.value,
      workflows.value,
      qrCodes.value,
      nextConnections ?? null
    );
    connections.value = nextState.connections;
    workflows.value = nextState.workflows;
    qrCodes.value = nextState.qrCodes;
    ready.value = nextState.ready;
  }

  function applyInitialSnapshot(nextConnections?: EndPointInfo[] | null, force = false): void {
    // HTTP 首屏快照只在实时快照尚未到达时兜底；force 用于 SSE 断开时回退到 HTTP 数据。
    if (ready.value && !force) return;
    replaceSnapshot(nextConnections);
  }

  function ensureInitialized(): void {
    if (initialized) return;
    initialized = true;

    realtimeStore.subscribeRealtimeEvent<ConnectionListPayload>('imconnection/list', payload => {
      replaceSnapshot(payload?.items ?? null);
    });

    realtimeStore.subscribeRealtimeEvent<ConnectionUpdatedPayload>(
      'imconnection/updated',
      payload => {
        connections.value = applyConnectionUpdate(connections.value, payload?.item ?? null);
      }
    );

    realtimeStore.subscribeRealtimeEvent<ConnectionWorkflowPayload>(
      'imconnection/workflow',
      payload => {
        if (!payload) return;
        workflows.value = applyConnectionWorkflow(
          workflows.value,
          payload.endpointId,
          payload.workflow ?? null
        );
      }
    );

    realtimeStore.subscribeRealtimeEvent<ConnectionQRCodePayload>(
      'imconnection/qrcode',
      payload => {
        if (!payload) return;
        qrCodes.value = applyConnectionQRCode(
          qrCodes.value,
          payload.endpointId,
          payload.img ?? null
        );
      }
    );

    watch(
      () => authStore.hasAccessToken,
      canAccess => {
        if (!canAccess) {
          connections.value = [];
          workflows.value = {};
          qrCodes.value = {};
          ready.value = false;
        }
      },
      { immediate: true }
    );

    // 全局 SSE 可能早于账号模块启动；重连以重新获取当前二维码和工作流快照。
    realtimeStore.reconnect();
  }

  const workflowOf = computed(() => (endpointId: string) => workflows.value[endpointId] ?? null);
  const qrCodeOf = computed(() => (endpointId: string) => qrCodes.value[endpointId] ?? '');

  return {
    connections,
    workflows,
    qrCodes,
    ready,
    workflowOf,
    qrCodeOf,
    applyInitialSnapshot,
    ensureInitialized,
  };
});
