import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import { useRealtimeClientStore } from '@/features/realtime/store';
import { applyLogAppend, applyLogSnapshot } from './logStreamState';
import type { BaseLogEntry, BaseLogItem } from './logStream';

type LogSnapshotPayload = {
  items?: BaseLogItem[] | null;
};

type LogAppendPayload = {
  item?: BaseLogItem | null;
};

export const useBaseLogStreamStore = defineStore('base-log-stream', () => {
  const authStore = useAuthStore(appPinia);
  const realtimeStore = useRealtimeClientStore(appPinia);

  const logs = ref<BaseLogEntry[]>([]);
  let initialized = false;

  function applySnapshot(items?: BaseLogItem[] | null): void {
    logs.value = applyLogSnapshot(logs.value, items ?? null);
  }

  function applyAppend(item?: BaseLogItem | null, limit = 500): void {
    logs.value = applyLogAppend(logs.value, item ?? null, limit);
  }

  function clearLogs(): void {
    logs.value = [];
  }

  function ensureInitialized(): void {
    // 首页日志是全局实时事件的业务投影，初始化只注册一次，避免多页面订阅重复写入。
    if (initialized) return;
    initialized = true;

    realtimeStore.subscribeRealtimeEvent<LogSnapshotPayload>('logs/snapshot', payload => {
      applySnapshot(payload?.items ?? null);
    });

    realtimeStore.subscribeRealtimeEvent<LogAppendPayload>('logs/append', payload => {
      applyAppend(payload?.item ?? null, 500);
    });

    watch(
      () => authStore.hasAccessToken,
      canAccess => {
        if (!canAccess) {
          clearLogs();
        }
      },
      { immediate: true }
    );

    // 全局 SSE 可能早于首页日志模块启动；重连以重新获取 logs/snapshot 快照。
    realtimeStore.reconnect();
  }

  const connected = computed(() => realtimeStore.connected);
  const errorText = computed(() => (realtimeStore.hasError ? '日志连接异常' : ''));
  const hasLogs = computed(() => logs.value.length > 0);

  function reconnect(): void {
    realtimeStore.reconnect();
  }

  function close(): void {
    realtimeStore.disconnect();
  }

  return {
    logs,
    connected,
    errorText,
    hasLogs,
    applySnapshot,
    applyAppend,
    clearLogs,
    ensureInitialized,
    reconnect,
    close,
  };
});
