import { computed, ref, watch } from 'vue';
import { hasAccessToken } from '@/features/auth/state';
import { subscribeRealtimeEvent, useRealtimeClient } from '@/features/realtime/client';
import { applyLogAppend, applyLogSnapshot } from './logStreamState';

export type BaseLogItem = {
  level: string;
  module?: string;
  ts: number;
  caller?: string;
  msg: string;
};

type LogSnapshotPayload = {
  items?: BaseLogItem[] | null;
};

type LogAppendPayload = {
  item?: BaseLogItem | null;
};

const logs = ref<BaseLogItem[]>([]);
let initialized = false;

// 首页日志流是实时事件的一个业务投影。
// 后端先推 logs/snapshot 作为当前缓冲，再持续推 logs/append；这里限制最多 500 条，
// 避免长时间打开首页导致表格渲染和内存无限增长。
function ensureInitialized(): void {
  if (initialized) return;
  initialized = true;

  subscribeRealtimeEvent<LogSnapshotPayload>('logs/snapshot', (payload) => {
    logs.value = applyLogSnapshot(logs.value, payload?.items ?? null);
  });

  subscribeRealtimeEvent<LogAppendPayload>('logs/append', (payload) => {
    logs.value = applyLogAppend(logs.value, payload?.item ?? null, 500);
  });

  watch(
    hasAccessToken,
    (canAccess) => {
      if (!canAccess) {
        logs.value = [];
      }
    },
    { immediate: true },
  );
}

export function useBaseLogStream() {
  const realtime = useRealtimeClient();
  ensureInitialized();

  return {
    logs,
    connected: realtime.connected,
    errorText: computed(() => (realtime.hasError.value ? '日志连接异常' : '')),
    hasLogs: computed(() => logs.value.length > 0),
    reconnect: realtime.reconnect,
    close: realtime.disconnect,
  };
}
