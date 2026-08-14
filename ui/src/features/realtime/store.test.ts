import { nextTick } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearAccessToken, finishAuthInitialization, setAccessToken } from '@/features/auth/state';
import { appPinia } from '@/pinia';
import { useRealtimeClientStore } from './store';

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
        : ((event: MessageEvent<string>) => listener.handleEvent(event));
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
        : ((event: MessageEvent<string>) => listener.handleEvent(event));
    handlers.delete(callback);
    if (handlers.size === 0) {
      this.listeners.delete(type);
    }
  }

  close(): void {}

  emitOpen(): void {
    this.onopen?.call(this as unknown as EventSource, new Event('open'));
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

  close(): void {}
}

describe('realtime client store', () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    FakeWebSocket.urls = [];
    vi.stubGlobal('EventSource', FakeEventSource);
    vi.stubGlobal('WebSocket', FakeWebSocket);
  });

  afterEach(() => {
    const store = useRealtimeClientStore(appPinia);
    store.disconnect();
    store.$dispose();
    clearAccessToken();
    vi.unstubAllGlobals();
  });

  it('uses EventSource as the only realtime transport when a token is available', async () => {
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
      'http://127.0.0.1/sd-api/v2/realtime/sse?token=token-realtime',
    );
    expect(store.activeTransport).toBe('sse');
  });
});
