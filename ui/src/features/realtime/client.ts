import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useRealtimeClientStore } from './store';

const realtimeStore = useRealtimeClientStore(appPinia);
const realtimeRefs = storeToRefs(realtimeStore);

export function subscribeRealtimeEvent<T = unknown>(
  event: string,
  handler: (payload: T) => void,
): () => void {
  return realtimeStore.subscribeRealtimeEvent(event, handler);
}

// 兼容层：旧代码仍通过 client.ts 读取实时状态，新代码优先直接使用 useRealtimeClientStore。
export function useRealtimeClient() {
  realtimeStore.ensureInitialized();

  return {
    connected: realtimeRefs.connected,
    connecting: realtimeRefs.connecting,
    activeTransport: realtimeRefs.activeTransport,
    hasError: realtimeRefs.hasError,
    lastError: realtimeRefs.lastError,
    reconnect: realtimeStore.reconnect,
    disconnect: realtimeStore.disconnect,
  };
}
