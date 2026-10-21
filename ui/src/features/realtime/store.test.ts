import { nextTick } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
import { useRealtimeClientStore } from './store';
const auth = useAuthStore(appPinia);
const { clearAccessToken, setAccessToken, finishInitialization: finishAuthInitialization } = auth;

class FakeEventSource {
  static instances: FakeEventSource[] = [];

  readonly url: string;
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null;
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null;

  private readonly listeners = new Map<string, Set<(event: MessageEvent<string>) => void>>();

  constructor(url: string | URL) {
    this.url = String(url);
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
    const callback =
      typeof listener === 'function'
        ? (listener as (event: MessageEvent<string>) => void)
        : (event: MessageEvent<string>) => listener.handleEvent(event);
    const handlers = this.listeners.get(type) ?? new Set<(event: MessageEvent<string>) => void>();
    handlers.add(callback);
    this.listeners.set(type, handlers);
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
    const handlers = this.listeners.get(type);
    if (!handlers) return;
    const callback =
      typeof listener === 'function'
        ? (listener as (event: MessageEvent<string>) => void)
        : (event: MessageEvent<string>) => listener.handleEvent(event);
    handlers.delete(callback);
    if (handlers.size === 0) {
      this.listeners.delete(type);
    }
  }

  close = vi.fn();

  emitOpen(): void {
    this.onopen?.call(this as unknown as EventSource, new Event('open'));
  }

  emitError(): void {
    this.onerror?.call(this as unknown as EventSource, new Event('error'));
  }

  emitMessage(type: string, data: unknown): void {
    const handlers = this.listeners.get(type);
    if (!handlers) return;
    const event = { data: JSON.stringify(data) } as MessageEvent<string>;
    for (const handler of handlers) {
      handler(event);
    }
  }
}

class FakeWebSocket {
  static urls: string[] = [];

  constructor(url: string | URL) {
    FakeWebSocket.urls.push(String(url));
  }

  close = vi.fn();
}

function emptyPollResponse() {
  return {
    status: 200,
    ok: true,
    json: async () => ({
      item: {
        epoch: 'test-epoch',
        logs: { items: [], latestSeq: 0, reset: false },
        connections: { items: [] },
        toolMessages: { items: [], latestSeq: 0, reset: false },
        timestamp: 1,
      },
    }),
  };
}

describe('realtime client store', () => {
  beforeEach(() => {
    auth.isInitializing = true;
    FakeEventSource.instances = [];
    FakeWebSocket.urls = [];
    vi.stubGlobal('EventSource', FakeEventSource);
    vi.stubGlobal('WebSocket', FakeWebSocket);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(emptyPollResponse()));
  });

  afterEach(() => {
    const store = useRealtimeClientStore(appPinia);
    store.disconnect();
    store.$dispose();
    clearAccessToken();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('释放 store 时关闭 SSE，之后的令牌变化不会重新连接', async () => {
    auth.setAccessToken('token-realtime');
    auth.finishInitialization();
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    await nextTick();
    const source = FakeEventSource.instances[0]!;
    store.$dispose();
    expect(source.close).toHaveBeenCalledOnce();
    auth.setAccessToken('next-token');
    await nextTick();
    expect(FakeEventSource.instances).toHaveLength(1);
  });

  it('polls with cursors and sends the token in the Authorization header', async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyPollResponse());
    vi.stubGlobal('fetch', fetchMock);
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();

    const snapshots: unknown[] = [];
    const appends: unknown[] = [];
    const toolMessages: unknown[] = [];
    store.subscribeRealtimeEvent('logs/snapshot', payload => snapshots.push(payload));
    store.subscribeRealtimeEvent('logs/append', payload => appends.push(payload));
    store.subscribeRealtimeEvent('tooltest/message', payload => toolMessages.push(payload));

    const pollItem = (logSeq: number, withAppend: boolean) => ({
      epoch: 'test-epoch',
      logs: {
        items: [{ seq: logSeq, level: 'info', module: 'core', ts: 1, msg: `log-${logSeq}` }],
        latestSeq: logSeq,
        reset: !withAppend,
      },
      connections: { items: [] },
      toolMessages: {
        items: [{ seq: logSeq, id: `tool-${logSeq}`, messageType: 'private' }],
        latestSeq: logSeq,
        reset: !withAppend,
      },
      timestamp: 1,
    });
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalled());
    fetchMock.mockClear();
    fetchMock.mockResolvedValueOnce({
      status: 200,
      ok: true,
      json: async () => ({ item: pollItem(1, false) }),
    });
    fetchMock.mockResolvedValueOnce({
      status: 200,
      ok: true,
      json: async () => ({ item: pollItem(2, true) }),
    });

    await vi.advanceTimersByTimeAsync(1000);
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    expect(store.activeTransport).toBe('polling');
    expect(snapshots).toHaveLength(1);
    expect(toolMessages).toHaveLength(1);
    const firstURL = String(fetchMock.mock.calls[0]?.[0]);
    expect(firstURL).toContain('afterLogSeq=0');
    expect(firstURL).toContain('afterToolSeq=0');
    expect(firstURL).not.toContain('token=');
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({
      headers: expect.objectContaining({ Authorization: 'Bearer token-realtime' }),
    });

    await vi.advanceTimersByTimeAsync(1000);
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));

    expect(appends).toHaveLength(1);
    expect(toolMessages).toHaveLength(2);
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain('afterLogSeq=1');
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain('afterToolSeq=1');
  });

  it('replays missed events through polling before switching back to SSE', async () => {
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();

    const source = FakeEventSource.instances[0]!;
    source.emitError();
    expect(store.activeTransport).toBe('polling');

    await vi.advanceTimersByTimeAsync(16000);
    expect(FakeEventSource.instances).toHaveLength(2);

    FakeEventSource.instances[1]!.emitOpen();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    expect(store.connected).toBe(true);
  });

  it('recovers log sequence gaps through polling and ignores stale snapshots', async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyPollResponse());
    vi.stubGlobal('fetch', fetchMock);
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    const source = FakeEventSource.instances[0]!;

    const appends: unknown[] = [];
    const snapshots: unknown[] = [];
    store.subscribeRealtimeEvent('logs/append', payload => appends.push(payload));
    store.subscribeRealtimeEvent('logs/snapshot', payload => snapshots.push(payload));

    source.emitOpen();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    fetchMock.mockClear();
    fetchMock.mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        item: {
          epoch: 'test-epoch',
          logs: {
            items: [
              { seq: 2, level: 'info', module: 'core', ts: 1, msg: 'log-2' },
              { seq: 3, level: 'info', module: 'core', ts: 1, msg: 'log-3' },
            ],
            latestSeq: 3,
            reset: false,
          },
          connections: { items: [] },
          toolMessages: { items: [], latestSeq: 0, reset: false },
          timestamp: 1,
        },
      }),
    });

    source.emitMessage('logs/append', {
      item: { seq: 1, level: 'info', module: 'core', ts: 1, msg: 'log-1' },
    });
    source.emitMessage('logs/append', {
      item: { seq: 3, level: 'info', module: 'core', ts: 1, msg: 'log-3' },
    });
    await vi.waitFor(() => expect(appends).toHaveLength(3));

    const messages = appends.map(payload => (payload as { item: { msg: string } }).item.msg);
    expect(messages).toEqual(['log-1', 'log-2', 'log-3']);

    source.emitMessage('logs/snapshot', {
      items: [{ seq: 2, level: 'info', module: 'core', ts: 1, msg: 'stale' }],
    });
    expect(snapshots).toHaveLength(0);
  });

  it('stops retrying SSE after three consecutive failures', async () => {
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();

    const fetchMock = vi.fn().mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        item: {
          logs: { items: [], latestSeq: 0, reset: false },
          connections: { items: [] },
          toolMessages: { items: [], latestSeq: 0, reset: false },
          timestamp: 1,
        },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    FakeEventSource.instances[0]!.emitError();
    await vi.waitFor(() => expect(store.activeTransport).toBe('polling'));

    for (let attempt = 1; attempt < 3; attempt += 1) {
      await vi.advanceTimersByTimeAsync(15000);
      expect(FakeEventSource.instances).toHaveLength(attempt + 1);
      FakeEventSource.instances[attempt]!.emitError();
    }

    await vi.advanceTimersByTimeAsync(30000);
    expect(FakeEventSource.instances).toHaveLength(3);
    expect(store.activeTransport).toBe('polling');
  });

  it('uses EventSource as the primary realtime transport when a token is available', async () => {
    setAccessToken('token-realtime');

    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    await nextTick();

    expect(FakeEventSource.instances).toHaveLength(0);

    finishAuthInitialization();
    await nextTick();

    expect(FakeWebSocket.urls).toEqual([]);
    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.instances[0]?.url).toBe(
      'http://127.0.0.1/sd-api/v2/realtime/sse?token=token-realtime'
    );
    expect(store.activeTransport).toBe('polling');

    FakeEventSource.instances[0]!.emitOpen();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    expect(store.connected).toBe(true);
  });
});
