import { defineStore } from 'pinia';
import { createParser } from 'eventsource-parser';
import { computed, onScopeDispose, shallowRef, watch } from 'vue';
import { getApiBaseUrl, joinApiBasePath, reportNetworkError } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';

type RealtimeEventHandler<T = unknown> = (payload: T) => void;
type RealtimeTransport = 'sse' | 'polling' | '';
type TimerHandle = ReturnType<typeof globalThis.setTimeout>;

type SequencedPayload = { seq?: number };
type LogSnapshotPayload = { items?: SequencedPayload[] | null };
type LogAppendPayload = { item?: SequencedPayload | null };
type ToolMessagePayload = SequencedPayload;
type PollData = {
  epoch?: string;
  logs?: {
    items?: SequencedPayload[] | null;
    latestSeq?: number;
    reset?: boolean;
  };
  connections?: { items?: unknown[] | null };
  toolMessages?: {
    items?: ToolMessagePayload[] | null;
    latestSeq?: number;
    reset?: boolean;
  };
};

const knownEventNames = [
  'system/ready',
  'system/heartbeat',
  'logs/snapshot',
  'logs/append',
  'imconnection/list',
  'imconnection/updated',
  'tooltest/message',
] as const;

const POLL_INTERVAL = 1000;
const SSE_RETRY_INTERVAL = 15000;
const SSE_CONNECT_TIMEOUT = 10000;
const SSE_STALL_TIMEOUT = 45000;
const SSE_MAX_FAILURES = 3;
const POLL_TIMEOUT = 8000;

function toSeq(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : 0;
}

export const useRealtimeClientStore = defineStore('realtime-client', () => {
  const authStore = useAuthStore(appPinia);

  const connected = shallowRef(false);
  const connecting = shallowRef(false);
  const lastError = shallowRef('');
  const activeTransport = shallowRef<RealtimeTransport>('');
  const hasError = computed(() => lastError.value !== '');

  const listeners = new Map<string, Set<RealtimeEventHandler>>();

  let sseController: AbortController | null = null;
  let reconnectTimer: TimerHandle | null = null;
  let pollTimer: TimerHandle | null = null;
  let sseConnectTimer: TimerHandle | null = null;
  let sseStallTimer: TimerHandle | null = null;
  let pollPromise: Promise<boolean> | null = null;
  let pollPromiseGeneration = -1;
  let pollingActive = false;
  let initialized = false;
  let connectGeneration = 0;
  let disconnectDialogShown = false;
  let sseFailureCount = 0;
  let serverEpoch = '';
  let logSeq = 0;
  let toolSeq = 0;
  let connectionVersion = 0;

  function dispatch(event: string, payload: unknown): void {
    const handlers = listeners.get(event);
    if (!handlers) return;
    for (const handler of handlers) {
      handler(payload);
    }
  }

  function dispatchRealtimeEvent(event: string, payload: unknown): void {
    if (event === 'logs/snapshot') {
      const items = (payload as LogSnapshotPayload | null)?.items ?? [];
      const snapshotSeq = items.reduce((latest, item) => Math.max(latest, toSeq(item?.seq)), 0);
      // 追赶轮询可能已经应用了比 SSE 首帧更新的日志。此时旧快照不能覆盖
      // 新状态，否则对应的 SSE 追加事件会被游标当作重复而丢弃。
      if (logSeq > 0 && snapshotSeq < logSeq) {
        return;
      }
      // 轮询切换期间只前进游标，避免与已应用的增量事件产生重复。
      logSeq = activeTransport.value === 'sse' ? snapshotSeq : Math.max(logSeq, snapshotSeq);
      dispatch(event, payload);
      return;
    }
    if (event === 'logs/append') {
      const seq = toSeq((payload as LogAppendPayload | null)?.item?.seq);
      if (seq > 0) {
        if (seq <= logSeq) return;
        if (logSeq > 0 && seq > logSeq + 1) {
          // 序号出现缺口，先通过轮询补齐，再按序应用后续事件。
          void pollOnce(connectGeneration);
          return;
        }
        logSeq = seq;
      }
      dispatch(event, payload);
      return;
    }
    if (event === 'tooltest/message') {
      const seq = toSeq((payload as ToolMessagePayload | null)?.seq);
      if (seq > 0) {
        if (seq <= toolSeq) return;
        if (toolSeq > 0 && seq > toolSeq + 1) {
          void pollOnce(connectGeneration);
          return;
        }
        toolSeq = seq;
      }
      dispatch(event, payload);
      return;
    }
    if (event === 'imconnection/list' || event === 'imconnection/updated') {
      connectionVersion += 1;
    }
    dispatch(event, payload);
  }

  function buildRealtimeURL(path: string, query: Record<string, string | number> = {}): string {
    // 测试环境没有 window，回退到固定 origin，避免仅因构造 URL 就抛错。
    const fallbackOrigin =
      typeof window === 'undefined' ? 'http://127.0.0.1' : window.location.origin;
    const url = new URL(joinApiBasePath(getApiBaseUrl() || fallbackOrigin, path));
    for (const [key, value] of Object.entries(query)) {
      url.searchParams.set(key, String(value));
    }
    return url.toString();
  }

  function clearWindowTimer(timer: TimerHandle | null): null {
    if (timer !== null) {
      globalThis.clearTimeout(timer);
    }
    return null;
  }

  function clearReconnectTimer(): void {
    reconnectTimer = clearWindowTimer(reconnectTimer);
  }

  function clearPollTimer(): void {
    pollTimer = clearWindowTimer(pollTimer);
  }

  function clearSSEWatchdogs(): void {
    sseConnectTimer = clearWindowTimer(sseConnectTimer);
    sseStallTimer = clearWindowTimer(sseStallTimer);
  }

  function closeSSE(): void {
    clearSSEWatchdogs();
    sseController?.abort();
    sseController = null;
  }

  function stopPolling(): void {
    pollingActive = false;
    clearPollTimer();
  }

  function cleanupTransports(): void {
    closeSSE();
    clearReconnectTimer();
    stopPolling();
    activeTransport.value = '';
  }

  function notifyDisconnect(): void {
    if (disconnectDialogShown) return;
    disconnectDialogShown = true;
    reportNetworkError();
  }

  function resetSSEWatchdog(generation: number, controller: AbortController): void {
    sseStallTimer = clearWindowTimer(sseStallTimer);
    sseStallTimer = globalThis.setTimeout(() => {
      if (generation !== connectGeneration || sseController !== controller) return;
      failSSE(generation);
    }, SSE_STALL_TIMEOUT);
  }

  function applyPollData(data: PollData, connectionsVersionAtRequest: number): void {
    const responseEpoch = typeof data.epoch === 'string' ? data.epoch : '';
    const epochChanged = responseEpoch !== '' && responseEpoch !== serverEpoch;
    if (responseEpoch !== '') {
      serverEpoch = responseEpoch;
    }
    if (epochChanged) {
      // 服务进程已重启，旧序号不再可信。
      logSeq = 0;
      toolSeq = 0;
    }

    const logs = data.logs;
    if (logs) {
      if (logs.reset) {
        dispatchRealtimeEvent('logs/snapshot', { items: logs.items ?? [] });
        logSeq = toSeq(logs.latestSeq);
      } else {
        for (const item of logs.items ?? []) {
          dispatchRealtimeEvent('logs/append', { item });
        }
        logSeq = Math.max(logSeq, toSeq(logs.latestSeq));
      }
    }

    // 如果本轮请求期间 SSE 已经推送了连接更新，轮询快照可能更旧，直接丢弃。
    if (connectionVersion === connectionsVersionAtRequest) {
      dispatchRealtimeEvent('imconnection/list', { items: data.connections?.items ?? [] });
    }

    const messages = data.toolMessages;
    if (messages) {
      const latestSeq = toSeq(messages.latestSeq);
      // 服务重启会让序号回退；此时旧游标不再适用，重新从缓冲区开始。
      if (messages.reset && latestSeq < toolSeq) {
        toolSeq = 0;
      }
      for (const message of messages.items ?? []) {
        dispatchRealtimeEvent('tooltest/message', message);
      }
      toolSeq = Math.max(toolSeq, latestSeq);
    }
  }

  async function doPoll(generation: number): Promise<boolean> {
    const controller = new AbortController();
    const timeout = globalThis.setTimeout(() => controller.abort(), POLL_TIMEOUT);
    const connectionsVersionAtRequest = connectionVersion;
    const token = authStore.currentAccessToken();
    try {
      const response = await fetch(
        buildRealtimeURL('/sd-api/v2/realtime/poll', {
          afterLogSeq: logSeq,
          afterToolSeq: toolSeq,
          afterEpoch: serverEpoch,
        }),
        {
          cache: 'no-store',
          headers: {
            Accept: 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          signal: controller.signal,
        }
      );
      if (generation !== connectGeneration || token !== authStore.currentAccessToken())
        return false;
      if (response.status === 401) {
        authStore.clearAccessToken();
        disconnect();
        return false;
      }
      if (!response.ok) {
        throw new Error(`poll failed: ${response.status}`);
      }
      const data = (await response.json()) as { item?: PollData } | PollData;
      if (generation !== connectGeneration || token !== authStore.currentAccessToken())
        return false;
      applyPollData(
        'item' in data && data.item ? data.item : (data as PollData),
        connectionsVersionAtRequest
      );
      if (pollingActive) {
        connected.value = true;
        connecting.value = false;
        lastError.value = '';
        disconnectDialogShown = false;
      }
      return true;
    } catch {
      if (generation !== connectGeneration || token !== authStore.currentAccessToken())
        return false;
      if (pollingActive) {
        connected.value = false;
        connecting.value = false;
        lastError.value = '实时连接异常';
        notifyDisconnect();
      }
      return false;
    } finally {
      globalThis.clearTimeout(timeout);
    }
  }

  function pollOnce(generation: number): Promise<boolean> {
    if (pollPromise && pollPromiseGeneration === generation) return pollPromise;
    const promise = doPoll(generation).finally(() => {
      if (pollPromise === promise) {
        pollPromise = null;
        pollPromiseGeneration = -1;
      }
    });
    pollPromise = promise;
    pollPromiseGeneration = generation;
    return promise;
  }

  function scheduleNextPoll(generation: number): void {
    if (!pollingActive || generation !== connectGeneration) return;
    clearPollTimer();
    pollTimer = globalThis.setTimeout(() => {
      pollTimer = null;
      void pollOnce(generation).finally(() => scheduleNextPoll(generation));
    }, POLL_INTERVAL);
  }

  function scheduleSSERetry(generation: number): void {
    clearReconnectTimer();
    if (!pollingActive || generation !== connectGeneration || sseFailureCount >= SSE_MAX_FAILURES) {
      return;
    }
    reconnectTimer = globalThis.setTimeout(() => {
      reconnectTimer = null;
      if (!pollingActive || generation !== connectGeneration) return;
      connectRealtime(generation, true);
    }, SSE_RETRY_INTERVAL);
  }

  function startPolling(generation: number): void {
    if (generation !== connectGeneration || pollingActive) return;
    closeSSE();
    pollingActive = true;
    activeTransport.value = 'polling';
    connecting.value = !connected.value;
    void pollOnce(generation).finally(() => scheduleNextPoll(generation));
    scheduleSSERetry(generation);
  }

  function activateSSE(): void {
    stopPolling();
    clearReconnectTimer();
    sseFailureCount = 0;
    activeTransport.value = 'sse';
    connected.value = true;
    connecting.value = false;
    lastError.value = '';
    disconnectDialogShown = false;
  }

  async function switchToSSE(generation: number, controller: AbortController): Promise<void> {
    // 切换前把轮询游标追平，覆盖 SSE 订阅建立前可能遗漏的最后一段事件。
    if (pollPromise && pollPromiseGeneration === generation) {
      await pollPromise;
    }
    if (generation !== connectGeneration || sseController !== controller) return;
    const drained = await pollOnce(generation);
    if (generation !== connectGeneration || sseController !== controller) return;
    if (!drained) {
      // 追赶失败时不能启用 SSE，否则未送达的工具回复会丢失。
      closeSSE();
      sseFailureCount += 1;
      scheduleSSERetry(generation);
      return;
    }
    activateSSE();
  }

  function failSSE(generation: number): void {
    if (generation !== connectGeneration) return;
    closeSSE();
    sseFailureCount += 1;
    if (!pollingActive) {
      connected.value = false;
      connecting.value = true;
      lastError.value = '实时连接异常';
      startPolling(generation);
      return;
    }
    scheduleSSERetry(generation);
  }

  function connectRealtime(generation: number, fromFallback: boolean): void {
    closeSSE();
    const controller = new AbortController();
    sseController = controller;
    if (!fromFallback) {
      connecting.value = true;
    }

    sseConnectTimer = globalThis.setTimeout(() => {
      if (generation !== connectGeneration || sseController !== controller) return;
      failSSE(generation);
    }, SSE_CONNECT_TIMEOUT);

    void streamRealtime(generation, controller, fromFallback);
  }

  async function streamRealtime(
    generation: number,
    controller: AbortController,
    fromFallback: boolean
  ): Promise<void> {
    const token = authStore.currentAccessToken();
    try {
      const response = await fetch(buildRealtimeURL('/sd-api/v2/realtime/sse'), {
        headers: {
          Accept: 'text/event-stream',
          Authorization: `Bearer ${token}`,
        },
        cache: 'no-store',
        signal: controller.signal,
      });
      if (
        generation !== connectGeneration ||
        sseController !== controller ||
        token !== authStore.currentAccessToken()
      ) {
        await response.body?.cancel();
        return;
      }
      if (response.status === 401) {
        authStore.clearAccessToken();
        disconnect();
        return;
      }
      if (
        !response.ok ||
        response.headers.get('content-type')?.split(';')[0]?.trim().toLowerCase() !==
          'text/event-stream' ||
        !response.body
      ) {
        await response.body?.cancel();
        throw new Error(`SSE connection failed: ${response.status}`);
      }

      sseConnectTimer = clearWindowTimer(sseConnectTimer);
      resetSSEWatchdog(generation, controller);
      if (fromFallback) {
        void switchToSSE(generation, controller);
      } else {
        activateSSE();
      }

      const parser = createParser({
        onEvent(event) {
          if (generation !== connectGeneration || sseController !== controller) return;
          const eventName = knownEventNames.find(name => name === event.event);
          if (!eventName) return;
          resetSSEWatchdog(generation, controller);
          let payload: unknown = null;
          if (event.data) {
            try {
              payload = JSON.parse(event.data);
            } catch {
              return;
            }
          }
          dispatchRealtimeEvent(eventName, payload);
        },
      });
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      try {
        while (!controller.signal.aborted) {
          const { done, value } = await reader.read();
          if (done) break;
          parser.feed(decoder.decode(value, { stream: true }));
        }
      } finally {
        await reader.cancel().catch(() => {});
        reader.releaseLock();
      }
    } catch {
      // 主动取消和旧连接的错误不会触发新会话的重连。
    }
    if (generation !== connectGeneration || sseController !== controller) return;
    failSSE(generation);
  }

  let stopAuthWatch: (() => void) | undefined;
  function ensureInitialized(): void {
    // 连接生命周期完全跟随 token：有 token 自动连接，没 token 立即断开并清理状态。
    if (initialized) return;
    initialized = true;

    stopAuthWatch = watch(
      () => (authStore.isInitializing ? '' : authStore.currentAccessToken()),
      token => {
        if (token) {
          reconnect();
        } else {
          disconnect();
          lastError.value = '';
        }
      },
      { immediate: true }
    );
  }

  function performReconnect(): void {
    if (!authStore.hasAccessToken || authStore.isInitializing) return;

    connectGeneration += 1;
    stopPolling();
    clearReconnectTimer();
    connected.value = false;
    connecting.value = true;
    lastError.value = '';
    sseFailureCount = 0;
    serverEpoch = '';
    connectionVersion = 0;
    logSeq = 0;
    toolSeq = 0;

    // 先轮询建立一条可用通道，再尝试 SSE；SSE 打开并完成追赶后切回。
    startPolling(connectGeneration);
    connectRealtime(connectGeneration, true);
  }

  function reconnect(): void {
    ensureInitialized();
    performReconnect();
  }

  function disconnect(): void {
    connectGeneration += 1;
    cleanupTransports();
    connected.value = false;
    connecting.value = false;
    disconnectDialogShown = false;
    sseFailureCount = 0;
    serverEpoch = '';
    connectionVersion = 0;
    logSeq = 0;
    toolSeq = 0;
  }

  function subscribeRealtimeEvent<T = unknown>(
    event: string,
    handler: RealtimeEventHandler<T>
  ): () => void {
    const handlers = listeners.get(event) ?? new Set<RealtimeEventHandler>();
    handlers.add(handler as RealtimeEventHandler);
    listeners.set(event, handlers);

    // 先登记监听器再启动连接，避免 SSE 首帧在业务订阅完成前到达。
    ensureInitialized();

    return () => {
      const current = listeners.get(event);
      if (!current) return;
      current.delete(handler as RealtimeEventHandler);
      if (current.size === 0) {
        listeners.delete(event);
      }
    };
  }

  onScopeDispose(() => {
    stopAuthWatch?.();
    disconnect();
    listeners.clear();
  });

  return {
    connected,
    connecting,
    lastError,
    activeTransport,
    hasError,
    reconnect,
    disconnect,
    subscribeRealtimeEvent,
    ensureInitialized,
  };
});
