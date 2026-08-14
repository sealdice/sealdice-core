import { defineStore } from 'pinia';
import { computed, shallowRef, watch } from 'vue';
import { getApiBaseUrl, joinApiBasePath, showNetworkErrorDialog } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';

type RealtimeEventHandler<T = unknown> = (payload: T) => void;

const knownEventNames = [
  'system/ready',
  'logs/snapshot',
  'logs/append',
  'imconnection/list',
  'imconnection/updated',
  'imconnection/workflow',
  'imconnection/qrcode',
  'tooltest/message',
] as const;

const RECONNECT_BASE_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;

export const useRealtimeClientStore = defineStore('realtime-client', () => {
  const authStore = useAuthStore(appPinia);

  const connected = shallowRef(false);
  const connecting = shallowRef(false);
  const lastError = shallowRef('');
  const activeTransport = shallowRef<'sse' | ''>('');
  const hasError = computed(() => lastError.value !== '');

  const listeners = new Map<string, Set<RealtimeEventHandler>>();

  let eventSource: EventSource | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;
  let initialized = false;
  let connectGeneration = 0;
  let manualDisconnect = false;
  let disconnectDialogShown = false;

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

    return url.toString();
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
    if (manualDisconnect || !authStore.hasAccessToken || authStore.isInitializing) return;
    if (typeof window === 'undefined') return;

    clearReconnectTimer();
    const delay = getReconnectDelay();
    reconnectTimer = window.setTimeout(() => {
      reconnectInternal();
    }, delay);
  }

  function closeSSE(): void {
    if (!eventSource) return;
    eventSource.onopen = null;
    eventSource.onerror = null;
    eventSource.close();
    eventSource = null;
  }

  function cleanupTransports(): void {
    closeSSE();
    activeTransport.value = '';
  }

  function connectRealtime(generation: number): void {
    // realtime 固定走 SSE；业务层只订阅事件，不需要感知底层传输细节。
    if (typeof EventSource === 'undefined') {
      connected.value = false;
      connecting.value = false;
      lastError.value = '浏览器不支持实时连接';
      notifyDisconnect();
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
      disconnectDialogShown = false;
    };

    source.onerror = () => {
      if (generation !== connectGeneration) return;
      connected.value = false;
      connecting.value = true;
      lastError.value = '实时连接异常';
      notifyDisconnect();
      closeSSE();
      scheduleReconnect();
    };

    for (const eventName of knownEventNames) {
      source.addEventListener(eventName, event => {
        if (generation !== connectGeneration) return;
        const messageEvent = event as MessageEvent<string>;
        let payload: unknown = null;
        if (messageEvent.data) {
          try {
            payload = JSON.parse(messageEvent.data);
          } catch {
            return;
          }
        }
        dispatch(eventName, payload);
      });
    }
  }

  function notifyDisconnect(): void {
    if (disconnectDialogShown) return;
    disconnectDialogShown = true;
    showNetworkErrorDialog();
  }

  function ensureInitialized(): void {
    // 连接生命周期完全跟随 token：有 token 自动连接，没 token 立即断开并清理状态。
    if (initialized) return;
    initialized = true;

    watch(
      () => authStore.hasAccessToken && !authStore.isInitializing,
      canAccess => {
        if (canAccess) {
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

    manualDisconnect = false;
    connectGeneration += 1;
    clearReconnectTimer();

    connected.value = false;
    connecting.value = true;
    lastError.value = '';

    connectRealtime(connectGeneration);
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
    disconnectDialogShown = false;
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
