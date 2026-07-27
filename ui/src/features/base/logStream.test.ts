import { it } from 'vitest';
import { appPinia } from '@/pinia';
import { useBaseLogStreamStore } from './logStreamStore';
import { useBaseLogStream } from './logStream';
import type { BaseLogItem } from './logStream';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) {
      throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
    }
  };

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
  assertEqual(stream.logs.value.length, 1);
  assertEqual(stream.hasLogs.value, true);

  store.applyAppend(second);
  assertEqual(stream.logs.value.length, 2);

  store.clearLogs();
  assertEqual(stream.logs.value.length, 0);
  assertEqual(stream.hasLogs.value, false);
});
