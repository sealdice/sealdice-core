import { nextTick } from 'vue';
import { it } from 'vitest';
import type { EndPointInfo } from '@/api';
import { clearAccessToken, setAccessToken } from '@/features/auth/state';
import { useRealtimeConnections } from './realtime';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) {
      throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
    }
  };

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
  assertEqual(realtimeConnections.connections.value.length, 1);
  assertEqual(realtimeConnections.ready.value, true);

  realtimeConnections.applyInitialSnapshot([]);
  assertEqual(realtimeConnections.connections.value.length, 1);

  setAccessToken('token-connect');
  await nextTick();
  clearAccessToken();
  await nextTick();

  assertEqual(realtimeConnections.connections.value.length, 0);
  assertEqual(realtimeConnections.ready.value, false);
});
