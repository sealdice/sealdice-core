import { nextTick } from 'vue';
import { expect, afterEach, beforeEach, it, vi } from 'vitest';
import type { EndPointInfo } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
const { clearAccessToken, setAccessToken } = useAuthStore(appPinia);
import { useRealtimeConnections } from './realtime';

class FakeEventSource {
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null;
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null;

  constructor(_url: string | URL) {
    void _url;
  }

  addEventListener(): void {}

  close(): void {}
}

beforeEach(() => {
  vi.stubGlobal('EventSource', FakeEventSource);
});

afterEach(() => {
  clearAccessToken();
  vi.unstubAllGlobals();
});

it('连接快照初始化及退出后的状态清理', async () => {
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
  };

  const realtimeConnections = useRealtimeConnections();
  realtimeConnections.applyInitialSnapshot([connection]);
  expect(realtimeConnections.connections.value.length).toBe(1);
  expect(realtimeConnections.ready.value).toBe(true);

  realtimeConnections.applyInitialSnapshot([]);
  expect(realtimeConnections.connections.value.length).toBe(1);

  setAccessToken('token-connect');
  await nextTick();
  clearAccessToken();
  await nextTick();

  expect(realtimeConnections.connections.value.length).toBe(0);
  expect(realtimeConnections.ready.value).toBe(false);
});
