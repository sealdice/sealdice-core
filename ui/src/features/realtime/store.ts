import { defineStore } from 'pinia';
import { computed, shallowRef, watch } from 'vue';
import { getApiBaseUrl, joinApiBasePath } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';

type RealtimeEventHandler<T = unknown> = (payload: T) => void;

type RealtimeEnvelope = {
  event?: string;
  payload?: unknown;
};

const knownEventNames = [
  'system/ready',
  'logs/snapshot',
  'logs/append',
  'imconnection/list',
  'imconnection/updated',
  'imconnection/workflow',
  'imconnection/qrcode',
] as const;

const RECONNECT_BASE_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;

export const useRealtimeClientStore = defineStore('realtime-client', () => {
  const authStore = useAuthStore(appPinia);

  const connected = shallowRef(false);
  const connecting = shallowRef(false);
  const lastError = shallowRef('');
  const activeTransport = shallowRef<'ws' | 'sse' | ''>('');
  const hasError = computed(() => lastError.value !== '');

  const listeners = new Map<string, Set<RealtimeEventHandler>>();

  let websocket: WebSocket | null = null;
  let eventSource: EventSource | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;
  let initialized = false;
  let connectGeneration = 0;
  let manualDisconnect = false;

  function dispatch(event: string, payload: unknown): void {
    const handlers = listeners.get(event);
    if (!handlers) return;
    for (const handler of handlers) {
      handler(payload);
    }
  }

  function buildRealtimeURL(path: string): string {
    // 测试环境没有 window，回退到固定 origin，避免仅因构造 URL 就抛错。
    const fallbackOrigin =
      typeof window === 'undefined' ? 'http://127.0.0.1' : window.location.origin;
    const url = new URL(joinApiBasePath(getApiBaseUrl() || fallbackOrigin, path));
    const token = authStore.currentAccessToken();
    if (token) {
      url.searchParams.set('token', token);
    }

    if (path.endsWith('/ws')) {
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    }

    return url.toString();
  }

  function parseEnvelope(raw: string): RealtimeEnvelope | null {
    try {
      return JSON.parse(raw) as RealtimeEnvelope;
    } catch {
      return null;
    }
  }

  function clearReconnectTimer(): void {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  }

  function getReconnectDelay(): number {
    const exponential = Math.min(RECONNECT_BASE_DELAY * 2 ** reconnectAttempt, RECONNECT_MAX_DELAY);
    reconnectAttempt += 1;
    const jitter = exponential * (0.8 + Math.random() * 0.4);
    return Math.round(jitter);
  }

  function resetReconnectAttempt(): void {
    reconnectAttempt = 0;
  }

  function scheduleReconnect(): void {
    if (manualDisconnect || !authStore.hasAccessToken) return;
    if (typeof window === 'undefined') return;

    clearReconnectTimer();
    const delay = getReconnectDelay();
    reconnectTimer = window.setTimeout(() => {
      reconnectInternal();
    }, delay);
  }

  function closeWS(): void {
    if (!websocket) return;
    websocket.onopen = null;
    websocket.onmessage = null;
    websocket.onerror = null;
    websocket.onclose = null;
    websocket.close();
    websocket = null;
  }

  function closeSSE(): void {
    if (!eventSource) return;
    eventSource.onopen = null;
    eventSource.onerror = null;
    eventSource.close();
    eventSource = null;
  }

  function cleanupTransports(): void {
    closeWS();
    closeSSE();
    activeTransport.value = '';
  }

  function connectSSE(generation: number): void {
    // SSE 是 WS 失败后的降级通道；业务层只订阅事件，不需要感知当前传输方式。
    if (typeof EventSource === 'undefined') {
      connected.value = false;
      connecting.value = false;
      lastError.value = '浏览器不支持实时连接';
      return;
    }

    cleanupTransports();
    activeTransport.value = 'sse';

    const source = new EventSource(buildRealtimeURL('/sd-api/v2/realtime/sse'));
    eventSource = source;

    source.onopen = () => {
      if (generation !== connectGeneration) return;
      resetReconnectAttempt();
      connected.value = true;
      connecting.value = false;
      lastError.value = '';
    };

    source.onerror = () => {
      if (generation !== connectGeneration) return;
      connected.value = false;
      connecting.value = true;
      lastError.value = '实时连接异常';
      closeSSE();
      scheduleReconnect();
    };

    for (const eventName of knownEventNames) {
      source.addEventListener(eventName, (event) => {
        if (generation !== connectGeneration) return;
        const messageEvent = event as MessageEvent<string>;
        const payload = messageEvent.data ? JSON.parse(messageEvent.data) : null;
        dispatch(eventName, payload);
      });
    }
  }

  function connectWS(generation: number): void {
    if (typeof WebSocket === 'undefined') {
      connectSSE(generation);
      return;
    }

    cleanupTransports();
    activeTransport.value = 'ws';

    const ws = new WebSocket(buildRealtimeURL('/sd-api/v2/realtime/ws'));
    websocket = ws;
    let opened = false;
    let fellBack = false;

    const fallbackToSSE = () => {
      if (fellBack || generation !== connectGeneration) return;
      fellBack = true;
      closeWS();
      connectSSE(generation);
    };

    ws.onopen = () => {
      if (generation !== connectGeneration) return;
      opened = true;
      resetReconnectAttempt();
      connected.value = true;
      connecting.value = false;
      lastError.value = '';
    };

    ws.onmessage = (event) => {
      if (generation !== connectGeneration) return;
      const envelope = parseEnvelope(String(event.data));
      if (!envelope?.event) return;
      dispatch(envelope.event, envelope.payload);
    };

    ws.onerror = () => {
      if (!opened) {
        fallbackToSSE();
        return;
      }
      if (generation !== connectGeneration) return;
      lastError.value = '实时连接异常';
    };

    ws.onclose = () => {
      if (!opened) {
        fallbackToSSE();
        return;
      }
      if (generation !== connectGeneration) return;
      connected.value = false;
      connecting.value = true;
      lastError.value = '实时连接已断开';
      scheduleReconnect();
    };
  }

  function ensureInitialized(): void {
    // 连接生命周期完全跟随 token：有 token 自动连接，没 token 立即断开并清理状态。
    if (initialized) return;
    initialized = true;

    watch(
      () => authStore.hasAccessToken,
      (canAccess) => {
        if (canAccess) {
          reconnect();
        } else {
          disconnect();
          lastError.value = '';
        }
      },
      { immediate: true },
    );
  }

  function performReconnect(): void {
    if (!authStore.hasAccessToken) return;

    manualDisconnect = false;
    connectGeneration += 1;
    clearReconnectTimer();

    connected.value = false;
    connecting.value = true;
    lastError.value = '';

    connectWS(connectGeneration);
  }

  function reconnect(): void {
    ensureInitialized();
    resetReconnectAttempt();
    performReconnect();
  }

  function reconnectInternal(): void {
    ensureInitialized();
    performReconnect();
  }

  function disconnect(): void {
    manualDisconnect = true;
    resetReconnectAttempt();
    clearReconnectTimer();
    cleanupTransports();
    connected.value = false;
    connecting.value = false;
  }

  function subscribeRealtimeEvent<T = unknown>(
    event: string,
    handler: RealtimeEventHandler<T>,
  ): () => void {
    // 订阅时顺便保证底层连接初始化，页面层只需要关心事件本身。
    ensureInitialized();

    const handlers = listeners.get(event) ?? new Set<RealtimeEventHandler>();
    handlers.add(handler as RealtimeEventHandler);
    listeners.set(event, handlers);

    return () => {
      const current = listeners.get(event);
      if (!current) return;
      current.delete(handler as RealtimeEventHandler);
      if (current.size === 0) {
        listeners.delete(event);
      }
    };
  }

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
