import { nextTick } from 'vue';
import { expect, afterEach, beforeEach, it, vi } from 'vitest';
import type { EndPointInfo } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
import { useRealtimeConnectionsStore } from './realtimeStore';

const { clearAccessToken, setAccessToken, finishInitialization } = useAuthStore(appPinia);
const sources: FakeEventSource[] = [];

class FakeEventSource {
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null;
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null;
  listeners = new Map<string, (event: MessageEvent) => void>();
  constructor(_url: string | URL) {
    void _url;
    sources.push(this);
  }
  addEventListener(name: string, handler: (event: MessageEvent) => void) {
    this.listeners.set(name, handler);
  }
  emit(name: string, payload: unknown) {
    this.listeners.get(name)?.(new MessageEvent(name, { data: JSON.stringify(payload) }));
  }
  close() {}
}

const connection: EndPointInfo = {
  adapter: {},
  cmdExecutedLastTime: 0,
  cmdExecutedNum: 0,
  enable: true,
  groupNum: 0,
  id: 'conn-1',
  isPublic: false,
  nickname: '测试连接',
  onlineTotalTime: 0,
  platform: 'QQ',
  protocolType: 'milky',
  relWorkDir: '',
  state: 2,
  userId: 'QQ:10001',
  workflow: { state: 'qrcode', hasQRCode: true, loginState: 2 },
  qrcode: 'data:image/png;base64,abc',
};

beforeEach(() => {
  sources.length = 0;
  vi.stubGlobal('EventSource', FakeEventSource);
  finishInitialization();
});

afterEach(async () => {
  clearAccessToken();
  await nextTick();
  vi.unstubAllGlobals();
});

it('HTTP 快照包含登录状态，迟到的初始快照不覆盖实时状态，退出后清空', async () => {
  const realtime = useRealtimeConnectionsStore(appPinia);
  realtime.ensureInitialized();
  realtime.applyInitialSnapshot([connection]);
  expect(realtime.connections).toHaveLength(1);
  expect(realtime.ready).toBe(true);
  expect(realtime.connections[0]?.workflow?.state).toBe('qrcode');
  expect(realtime.connections[0]?.qrcode).toBe(connection.qrcode);

  realtime.applyInitialSnapshot([]);
  expect(realtime.connections).toHaveLength(1);

  setAccessToken('token-connect');
  await nextTick();
  clearAccessToken();
  await nextTick();
  expect(realtime.connections).toEqual([]);
  expect(realtime.ready).toBe(false);
});

it('一条 SSE 更新同时更新连接和登录状态，列表删除同时清除二维码', async () => {
  setAccessToken('token-connect');
  const realtime = useRealtimeConnectionsStore(appPinia);
  realtime.ensureInitialized();
  await nextTick();
  const source = sources.at(-1)!;

  source.emit('imconnection/list', { items: [connection] });
  expect(realtime.connections[0]?.qrcode).toBe(connection.qrcode);

  source.emit('imconnection/updated', {
    item: {
      ...connection,
      state: 1,
      qrcode: '',
      workflow: { state: 'success', hasQRCode: false, loginState: 3 },
    },
  });
  expect(realtime.connections[0]?.state).toBe(1);
  expect(realtime.connections[0]?.workflow?.state).toBe('success');
  expect(realtime.connections[0]?.qrcode).toBe('');

  source.emit('imconnection/list', { items: [] });
  expect(realtime.connections).toEqual([]);
});
