import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';
import { useBaseLogStreamStore } from './logStreamStore';
import { useBaseLogStream } from './logStream';
import type { BaseLogItem } from './logStream';

it('快照、追加和清空更新可见日志', async () => {
  const first: BaseLogItem = {
    level: 'info',
    msg: 'first',
    ts: 1,
  };
  const second: BaseLogItem = {
    level: 'warn',
    msg: 'second',
    ts: 2,
  };

  const store = useBaseLogStreamStore(appPinia);
  const stream = useBaseLogStream();
  store.clearLogs();

  store.applySnapshot([first]);
  expect(stream.logs.value.length).toBe(1);
  expect(stream.hasLogs.value).toBe(true);

  store.applyAppend(second);
  expect(stream.logs.value.length).toBe(2);

  store.clearLogs();
  expect(stream.logs.value.length).toBe(0);
  expect(stream.hasLogs.value).toBe(false);
});
