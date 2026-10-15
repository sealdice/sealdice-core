import { storeToRefs } from 'pinia';
import { useRealtimeClient } from '@/features/realtime/client';
import { useRealtimeConnectionsStore } from './realtimeStore';

// 在消费时组合业务状态与传输状态，模块加载不触发全局初始化。
export function useRealtimeConnections() {
  const realtimeConnectionsStore = useRealtimeConnectionsStore();
  const realtimeConnectionRefs = storeToRefs(realtimeConnectionsStore);
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
