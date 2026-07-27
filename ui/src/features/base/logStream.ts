import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useBaseLogStreamStore } from './logStreamStore';

export type BaseLogItem = {
  level: string;
  module?: string;
  ts: number;
  caller?: string;
  msg: string;
};

const baseLogStreamStore = useBaseLogStreamStore(appPinia);
const baseLogStreamRefs = storeToRefs(baseLogStreamStore);

export function useBaseLogStream() {
  baseLogStreamStore.ensureInitialized();

  return {
    // 兼容层：旧页面继续用 useBaseLogStream()，内部日志状态已经迁到 Pinia。
    logs: baseLogStreamRefs.logs,
    connected: baseLogStreamRefs.connected,
    errorText: baseLogStreamRefs.errorText,
    hasLogs: baseLogStreamRefs.hasLogs,
    reconnect: baseLogStreamStore.reconnect,
    close: baseLogStreamStore.close,
  };
}
