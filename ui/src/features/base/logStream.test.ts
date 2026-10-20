import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';
import { useBaseLogStreamStore } from './logStreamStore';
import type { BaseLogItem } from './logStreamState';

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
  store.clearLogs();

  store.applySnapshot([first]);
  expect(store.logs.length).toBe(1);

  store.applyAppend(second);
  expect(store.logs.length).toBe(2);

  store.clearLogs();
  expect(store.logs.length).toBe(0);
});
