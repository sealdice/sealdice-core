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

async function openSSE() {
  setAccessToken('token-realtime');
  const store = useRealtimeClientStore(appPinia);
  store.ensureInitialized();
  finishAuthInitialization();
  await nextTick();
  const request = sseRequests.at(-1)!;
  request.open();
  await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
  return { store, request };
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
    store.subscribeRealtimeEvent('logs/append', payload => appends.push(payload));
    store.subscribeRealtimeEvent('logs/snapshot', payload => snapshots.push(payload));

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
    await vi.waitFor(() => expect(snapshots).toHaveLength(0));
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

  it('parses split UTF-8, CRLF, multiline data and multiple events from the response stream', async () => {
    const { store, request } = await openSSE();
    const messages: unknown[] = [];
    store.subscribeRealtimeEvent('tooltest/message', payload => messages.push(payload));
    const bytes = new TextEncoder().encode(
      ': heartbeat\r\nevent: tooltest/message\r\ndata: {"seq":1,\r\ndata: "text":"你好"}\r\n\r\n'
    );
    for (const byte of bytes) request.push(new Uint8Array([byte]));
    request.push(
      'event: tooltest/message\ndata: invalid-json\n\n' +
        'event: tooltest/message\ndata: {"seq":2,"text":"next"}\n\n' +
        'event: tooltest/message\ndata: {"seq":3,"text":"last"}\n\n'
    );
    await vi.waitFor(() =>
      expect(messages).toEqual([
        { seq: 1, text: '你好' },
        { seq: 2, text: 'next' },
        { seq: 3, text: 'last' },
      ])
    );
  });

  it('clears the session on SSE 401 and stops reconnecting', async () => {
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    const request = sseRequests[0]!;
    request.open(401, 'application/json');
    await vi.waitFor(() => expect(auth.hasAccessToken).toBe(false));
    expect(request.signal.aborted).toBe(true);
    expect(store.activeTransport).toBe('');
    await vi.advanceTimersByTimeAsync(60000);
    expect(sseRequests).toHaveLength(1);
  });

  it.each([
    [503, 'application/json'],
    [200, 'text/html'],
  ])('keeps polling when SSE responds with %s %s', async (status, contentType) => {
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    const request = sseRequests[0]!;
    request.open(status, contentType);
    await vi.waitFor(() => expect(request.cancelled).toBe(true));
    expect(store.activeTransport).toBe('polling');
    expect(auth.currentAccessToken()).toBe('token-realtime');
  });

  it('falls back to polling and retries once when an established SSE stream ends', async () => {
    vi.useFakeTimers();
    const { store, request } = await openSSE();
    request.end();
    await vi.waitFor(() => expect(store.activeTransport).toBe('polling'));
    expect(request.signal.aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(15000);
    expect(sseRequests).toHaveLength(2);
  });

  it('aborts an SSE connection that never returns response headers', async () => {
    vi.useFakeTimers();
    setAccessToken('token-realtime');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    const request = sseRequests[0]!;
    await vi.advanceTimersByTimeAsync(10000);
    expect(request.signal.aborted).toBe(true);
    expect(store.activeTransport).toBe('polling');
    await vi.advanceTimersByTimeAsync(15000);
    expect(sseRequests).toHaveLength(2);
  });

  it('keeps SSE alive on heartbeats and falls back when the stream stalls', async () => {
    vi.useFakeTimers();
    const { store, request } = await openSSE();
    await vi.advanceTimersByTimeAsync(30000);
    request.send('system/heartbeat', {});
    await vi.advanceTimersByTimeAsync(30000);
    expect(store.activeTransport).toBe('sse');
    await vi.advanceTimersByTimeAsync(15000);
    expect(store.activeTransport).toBe('polling');
    expect(request.signal.aborted).toBe(true);
  });

  it('aborts SSE on logout without starting another connection', async () => {
    vi.useFakeTimers();
    const { store, request } = await openSSE();
    clearAccessToken();
    await nextTick();
    expect(request.signal.aborted).toBe(true);
    expect(store.connected).toBe(false);
    expect(store.activeTransport).toBe('');
    await vi.advanceTimersByTimeAsync(60000);
    expect(sseRequests).toHaveLength(1);
  });

  it('replaces the SSE connection when the token changes', async () => {
    const { request } = await openSSE();
    setAccessToken('next-token');
    await nextTick();
    expect(request.signal.aborted).toBe(true);
    expect(sseRequests).toHaveLength(2);
    expect(sseRequests[1]?.options.headers).toMatchObject({ Authorization: 'Bearer next-token' });
    expect(sseRequests[1]?.url).not.toContain('token');
  });

  it('ignores a delayed SSE 401 from the previous session', async () => {
    let resolveOld!: (response: Response) => void;
    const oldResponse = new Promise<Response>(resolve => {
      resolveOld = resolve;
    });
    const fetchMock = vi.mocked(fetch);
    const transport = fetchMock.getMockImplementation()!;
    fetchMock.mockImplementation((input, options) => {
      if (
        String(input).endsWith('/realtime/sse') &&
        new Headers(options?.headers).get('Authorization') === 'Bearer old-token'
      ) {
        return oldResponse;
      }
      return transport(input, options);
    });
    setAccessToken('old-token');
    const store = useRealtimeClientStore(appPinia);
    store.ensureInitialized();
    finishAuthInitialization();
    await nextTick();
    setAccessToken('next-token');
    await nextTick();
    resolveOld(new Response('', { status: 401 }));
    sseRequests[0]!.open();
    await vi.waitFor(() => expect(store.activeTransport).toBe('sse'));
    expect(auth.currentAccessToken()).toBe('next-token');
  });
});
