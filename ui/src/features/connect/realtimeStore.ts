import { defineStore } from 'pinia';
import { computed, onScopeDispose, ref, shallowRef, watch } from 'vue';
import type { EndPointInfo, WorkflowResp } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import { useRealtimeClientStore } from '@/features/realtime/store';

type ConnectionListPayload = {
  items?: EndPointInfo[] | null;
};

type ConnectionUpdatedPayload = {
  item?: EndPointInfo | null;
};

export const useRealtimeConnectionsStore = defineStore('connect-realtime', () => {
  const authStore = useAuthStore(appPinia);
  const realtimeStore = useRealtimeClientStore(appPinia);

  const connections = ref<EndPointInfo[]>([]);
  const workflows = computed<Record<string, WorkflowResp>>(() =>
    Object.fromEntries(
      connections.value.flatMap(item => (item.workflow ? [[item.id, item.workflow]] : []))
    )
  );
  const qrCodes = computed<Record<string, string>>(() =>
    Object.fromEntries(
      connections.value.flatMap(item => (item.qrcode ? [[item.id, item.qrcode]] : []))
    )
  );
  const ready = shallowRef(false);

  let initialized = false;
  const disposers: Array<() => void> = [];
  onScopeDispose(() => disposers.forEach(dispose => dispose()));

  function replaceSnapshot(nextConnections?: EndPointInfo[] | null): void {
    connections.value = nextConnections ? [...nextConnections] : [];
    ready.value = true;
  }

  function applyInitialSnapshot(nextConnections?: EndPointInfo[] | null, force = false): void {
    // HTTP 首屏快照只在实时快照尚未到达时兜底；force 用于 SSE 断开时回退到 HTTP 数据。
    if (ready.value && !force) return;
    replaceSnapshot(nextConnections);
  }

  function ensureInitialized(): void {
    if (initialized) return;
    initialized = true;

    disposers.push(
      realtimeStore.subscribeRealtimeEvent<ConnectionListPayload>('imconnection/list', payload => {
        replaceSnapshot(payload?.items ?? null);
      }),
      realtimeStore.subscribeRealtimeEvent<ConnectionUpdatedPayload>(
        'imconnection/updated',
        payload => {
          const item = payload?.item;
          if (!item) return;
          const index = connections.value.findIndex(connection => connection.id === item.id);
          if (index < 0) connections.value.push(item);
          else connections.value[index] = item;
        }
      ),
      watch(
        () => authStore.hasAccessToken,
        canAccess => {
          if (!canAccess) {
            connections.value = [];
            ready.value = false;
          }
        },
        { immediate: true }
      )
    );

    // 业务订阅可能晚于全局 SSE；重连以接收完整快照。
    realtimeStore.reconnect();
  }

  return {
    connections,
    workflows,
    qrCodes,
    ready,
    applyInitialSnapshot,
    ensureInitialized,
  };
});
