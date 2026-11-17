import { nextTick } from 'vue';
import { expect, afterEach, beforeEach, it, vi } from 'vitest';
import type { EndPointInfo } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
import { useRealtimeConnectionsStore } from './realtimeStore';
import { SSERequest } from '@/features/realtime/sseTestUtils';
import { useRealtimeClientStore } from '@/features/realtime/store';

const { clearAccessToken, setAccessToken, finishInitialization } = useAuthStore(appPinia);
const sources: SSERequest[] = [];

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
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, options: RequestInit) => {
      if (new URL(url).pathname.endsWith('/realtime/sse')) {
        const source = new SSERequest(url, options);
        sources.push(source);
        return source.response;
      }
      return Promise.resolve(Response.json({ item: { connections: { items: [] } } }));
    })
  );
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
  source.open();
  await vi.waitFor(() => expect(useRealtimeClientStore(appPinia).activeTransport).toBe('sse'));

  source.send('imconnection/list', { items: [connection] });
  await vi.waitFor(() => expect(realtime.connections).toHaveLength(1));
  expect(realtime.connections[0]?.qrcode).toBe(connection.qrcode);

  source.send('imconnection/updated', {
    item: {
      ...connection,
      state: 1,
      qrcode: '',
      workflow: { state: 'success', hasQRCode: false, loginState: 3 },
    },
  });
  await vi.waitFor(() => expect(realtime.connections[0]?.state).toBe(1));
  expect(realtime.connections[0]?.workflow?.state).toBe('success');
  expect(realtime.connections[0]?.qrcode).toBe('');

  source.send('imconnection/list', { items: [] });
  await vi.waitFor(() => expect(realtime.connections).toEqual([]));
});
