import { computed, ref, watch } from 'vue';
import { getApiBaseUrl } from '@/api';
import { clearAccessToken, currentAccessToken, hasAccessToken } from '@/features/auth/state';
import { queryClient } from '@/queryClient';

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

const connected = ref(false);
const connecting = ref(false);
const lastError = ref('');
const activeTransport = ref<'ws' | 'sse' | ''>('');

const listeners = new Map<string, Set<RealtimeEventHandler>>();

let websocket: WebSocket | null = null;
let eventSource: EventSource | null = null;
let reconnectTimer: number | null = null;
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
  const url = new URL(path, getApiBaseUrl() || window.location.origin);
  const token = currentAccessToken();
  if (token) {
    url.searchParams.set('token', token);
  }

  if (path.endsWith('/ws')) {
    url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
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

function clearSessionForUnauthorized(): void {
  clearAccessToken();
  queryClient.clear();
}

function clearReconnectTimer(): void {
  if (reconnectTimer !== null) {
    window.clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function scheduleReconnect(): void {
  if (manualDisconnect || !hasAccessToken.value) return;
  clearReconnectTimer();
  reconnectTimer = window.setTimeout(() => {
    reconnect();
  }, 1500);
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
    if (generation != connectGeneration) return;
    connected.value = true;
    connecting.value = false;
    lastError.value = '';
  };

  source.onerror = () => {
    if (generation != connectGeneration) return;
    connected.value = false;
    connecting.value = true;
    lastError.value = '实时连接异常';
  };

  for (const eventName of knownEventNames) {
    source.addEventListener(eventName, (event) => {
      if (generation != connectGeneration) return;
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
    if (fellBack || generation != connectGeneration) return;
    fellBack = true;
    closeWS();
    connectSSE(generation);
  };

  ws.onopen = () => {
    if (generation != connectGeneration) return;
    opened = true;
    connected.value = true;
    connecting.value = false;
    lastError.value = '';
  };

  ws.onmessage = (event) => {
    if (generation != connectGeneration) return;
    const envelope = parseEnvelope(String(event.data));
    if (!envelope?.event) return;
    dispatch(envelope.event, envelope.payload);
  };

  ws.onerror = () => {
    if (!opened) {
      fallbackToSSE();
      return;
    }
    if (generation != connectGeneration) return;
    lastError.value = '实时连接异常';
  };

  ws.onclose = () => {
    if (!opened) {
      fallbackToSSE();
      return;
    }
    if (generation != connectGeneration) return;
    connected.value = false;
    connecting.value = true;
    lastError.value = '实时连接已断开';
    scheduleReconnect();
  };
}

function ensureInitialized(): void {
  if (initialized) return;
  initialized = true;

  watch(
    hasAccessToken,
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

function reconnect(): void {
  ensureInitialized();
  if (!hasAccessToken.value) return;

  manualDisconnect = false;
  connectGeneration += 1;
  clearReconnectTimer();

  connected.value = false;
  connecting.value = true;
  lastError.value = '';

  connectWS(connectGeneration);
}

function disconnect(): void {
  manualDisconnect = true;
  clearReconnectTimer();
  cleanupTransports();
  connected.value = false;
  connecting.value = false;
}

export function subscribeRealtimeEvent<T = unknown>(
  event: string,
  handler: RealtimeEventHandler<T>,
): () => void {
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

export function useRealtimeClient() {
  ensureInitialized();

  watch(lastError, (message) => {
    if (message === 'unauthorized') {
      clearSessionForUnauthorized();
    }
  });

  return {
    connected,
    connecting,
    activeTransport,
    hasError: computed(() => lastError.value !== ''),
    lastError,
    reconnect,
    disconnect,
  };
}
