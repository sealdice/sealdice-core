import { computed, ref, watch } from 'vue';
import { getApiBaseUrl, joinApiBasePath } from '@/api';
import { currentAccessToken, hasAccessToken } from '@/features/auth/state';

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
// 自动重连采用指数退避 + 抖动，避免后端不可用时形成固定间隔的重连风暴。
const RECONNECT_BASE_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;

let reconnectTimer: number | null = null;
let reconnectAttempt = 0;
let initialized = false;
let connectGeneration = 0;
let manualDisconnect = false;

// 实时层是全局事件总线，不直接保存业务状态。
// 业务 feature 通过 subscribeRealtimeEvent 订阅事件，再把 payload 转成自己的 ref。
function dispatch(event: string, payload: unknown): void {
  const handlers = listeners.get(event);
  if (!handlers) return;
  for (const handler of handlers) {
    handler(payload);
  }
}

function buildRealtimeURL(path: string): string {
  // WebSocket/EventSource 不能统一注入 Authorization header，因此实时接口使用 query token。
  // token 仍来自 features/auth/state.ts，是同一个 V2 token 源。
  const url = new URL(joinApiBasePath(getApiBaseUrl() || window.location.origin, path));
  const token = currentAccessToken();
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
  // 指数退避到上限后保持该间隔持续重连，既防止风暴又能自愈。
  const exponential = Math.min(RECONNECT_BASE_DELAY * 2 ** reconnectAttempt, RECONNECT_MAX_DELAY);
  reconnectAttempt += 1;
  // ±20% 抖动，避免多个客户端同步重连形成惊群。
  const jitter = exponential * (0.8 + Math.random() * 0.4);
  return Math.round(jitter);
}

function resetReconnectAttempt(): void {
  reconnectAttempt = 0;
}

function scheduleReconnect(): void {
  // 自动重连只在“非手动断开且仍有 token”时发生，避免退出登录后后台继续重连。
  if (manualDisconnect || !hasAccessToken.value) return;
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
  // SSE 是 WS 不可用时的降级通道。两种传输都派发同一批事件名，
  // 所以业务订阅方不需要关心当前 activeTransport。
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
    resetReconnectAttempt();
    connected.value = true;
    connecting.value = false;
    lastError.value = '';
  };

  source.onerror = () => {
    if (generation != connectGeneration) return;
    connected.value = false;
    connecting.value = true;
    lastError.value = '实时连接异常';
    // 主动关闭由浏览器原生重连的通道，交给应用层统一退避重连。
    closeSSE();
    scheduleReconnect();
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
  // connectGeneration 用来丢弃旧连接的异步回调，防止快速登录/登出/重连时
  // 上一代 socket 把状态写回当前 UI。
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
    resetReconnectAttempt();
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
  // 通过 token 状态驱动连接生命周期：有 token 自动连接，无 token 断开并清状态。
  // 这样页面只要订阅事件，不需要知道何时建立底层连接。
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

function performReconnect(): void {
  if (!hasAccessToken.value) return;

  manualDisconnect = false;
  connectGeneration += 1;
  clearReconnectTimer();

  connected.value = false;
  connecting.value = true;
  lastError.value = '';

  connectWS(connectGeneration);
}

function reconnect(): void {
  // 手动重连视为全新尝试，重置退避计数，让用户点“重连”后立即用最短间隔。
  ensureInitialized();
  resetReconnectAttempt();
  performReconnect();
}

function reconnectInternal(): void {
  // 自动重连保留退避计数，使连续失败时间隔持续增长。
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

export function subscribeRealtimeEvent<T = unknown>(
  event: string,
  handler: RealtimeEventHandler<T>,
): () => void {
  // 返回 unsubscribe，页面级订阅必须在 onBeforeUnmount 调用；
  // feature 级单例订阅则由 initialized guard 控制只注册一次。
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
