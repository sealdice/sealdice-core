import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useRealtimeClientStore } from './store';

// 无组件上下文的订阅方显式使用应用 Pinia；导入模块本身不创建 store。
export function subscribeRealtimeEvent<T>(
  event: string,
  handler: (payload: T) => void
): () => void {
  return useRealtimeClientStore(appPinia).subscribeRealtimeEvent(event, handler);
}

export function useRealtimeClient() {
  const store = useRealtimeClientStore();
  store.ensureInitialized();
  return { ...storeToRefs(store), reconnect: store.reconnect, disconnect: store.disconnect };
}
