import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useRealtimeClient } from '@/features/realtime/client';
import { useRealtimeConnectionsStore } from './realtimeStore';

const realtimeConnectionsStore = useRealtimeConnectionsStore(appPinia);
const realtimeConnectionRefs = storeToRefs(realtimeConnectionsStore);

// 兼容层：旧代码继续用 useRealtimeConnections()，内部状态已经迁到 Pinia。
export function useRealtimeConnections() {
  realtimeConnectionsStore.ensureInitialized();
  const realtime = useRealtimeClient();

  return {
    connections: realtimeConnectionRefs.connections,
    workflows: realtimeConnectionRefs.workflows,
    qrCodes: realtimeConnectionRefs.qrCodes,
    ready: realtimeConnectionRefs.ready,
    connected: realtime.connected,
    connecting: realtime.connecting,
    lastError: realtime.lastError,
    reconnect: realtime.reconnect,
    applyInitialSnapshot: realtimeConnectionsStore.applyInitialSnapshot,
    workflowOf: realtimeConnectionRefs.workflowOf,
    qrCodeOf: realtimeConnectionRefs.qrCodeOf,
  };
}
