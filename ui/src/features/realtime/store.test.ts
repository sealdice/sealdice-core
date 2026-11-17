import { nextTick } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
import { useRealtimeClientStore } from './store';
import { SSERequest } from './sseTestUtils';
const auth = useAuthStore(appPinia);
const { clearAccessToken, setAccessToken, finishInitialization: finishAuthInitialization } = auth;

const sseRequests: SSERequest[] = [];
const pollFetch = vi.fn();

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
    sseRequests.length = 0;
    pollFetch.mockReset().mockResolvedValue(emptyPollResponse());
    vi.stubGlobal(
      'fetch',
      vi.fn((input: string, options: RequestInit) => {
        if (new URL(input).pathname.endsWith('/realtime/sse')) {
          const request = new SSERequest(input, options);
          sseRequests.push(request);
          return request.response;
        }
        return pollFetch(input, options);
      })
    );
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
    const source = sseRequests[0]!;
    store.$dispose();
    expect(source.signal.aborted).toBe(true);
    auth.setAccessToken('next-token');
    await nextTick();
    expect(sseRequests).toHaveLength(1);
  });

  it('polls with cursors and sends the token in the Authorization header', async () => {
    const fetchMock = pollFetch;
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

    const source = sseRequests[0]!;
    source.fail();
    expect(store.activeTransport).toBe('polling');

    await vi.advanceTimersByTimeAsync(16000);
    expect(sseRequests).toHaveLength(2);

    sseRequests[1]!.open();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    expect(store.connected).toBe(true);
  });

  it('recovers log sequence gaps through polling and ignores stale snapshots', async () => {
    const fetchMock = pollFetch;
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    const source = sseRequests[0]!;

    const appends: unknown[] = [];
    const snapshots: unknown[] = [];
    const heartbeat = vi.fn();
    store.subscribeRealtimeEvent('logs/append', payload => appends.push(payload));
    store.subscribeRealtimeEvent('logs/snapshot', payload => snapshots.push(payload));
    store.subscribeRealtimeEvent('system/heartbeat', heartbeat);

    source.open();
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

    source.send('logs/append', {
      item: { seq: 1, level: 'info', module: 'core', ts: 1, msg: 'log-1' },
    });
    source.send('logs/append', {
      item: { seq: 3, level: 'info', module: 'core', ts: 1, msg: 'log-3' },
    });
    await vi.waitFor(() => expect(appends).toHaveLength(3));

    const messages = appends.map(payload => (payload as { item: { msg: string } }).item.msg);
    expect(messages).toEqual(['log-1', 'log-2', 'log-3']);

    source.send('logs/snapshot', {
      items: [{ seq: 2, level: 'info', module: 'core', ts: 1, msg: 'stale' }],
    });
    source.send('system/heartbeat', {});
    await vi.waitFor(() => expect(heartbeat).toHaveBeenCalledOnce());
    expect(snapshots).toHaveLength(0);
  });

  it('stops retrying SSE after three consecutive failures', async () => {
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();

    pollFetch.mockResolvedValue({
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

    sseRequests[0]!.fail();
    await vi.waitFor(() => expect(store.activeTransport).toBe('polling'));

    for (let attempt = 1; attempt < 3; attempt += 1) {
      await vi.advanceTimersByTimeAsync(15000);
      expect(sseRequests).toHaveLength(attempt + 1);
      sseRequests[attempt]!.fail();
    }

    await vi.advanceTimersByTimeAsync(30000);
    expect(sseRequests).toHaveLength(3);
    expect(store.activeTransport).toBe('polling');
  });

  it('authenticates SSE through the Authorization header without putting the token in the URL', async () => {
    setAccessToken('token-realtime');

    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    await nextTick();

    expect(sseRequests).toHaveLength(0);

    finishAuthInitialization();
    await nextTick();

    expect(sseRequests).toHaveLength(1);
    expect(sseRequests[0]?.url).toBe('http://127.0.0.1/sd-api/v2/realtime/sse');
    expect(sseRequests[0]?.options.headers).toEqual({
      Accept: 'text/event-stream',
      Authorization: 'Bearer token-realtime',
    });
    expect(store.activeTransport).toBe('polling');

    sseRequests[0]!.open();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    expect(store.connected).toBe(true);
  });
});
