import type { BaseLogItem } from './logStream';
import { applyLogAppend, applyLogSnapshot, resetLogIdSequence } from './logStreamState';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
      throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
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

  resetLogIdSequence();

  assertDeepEqual(applyLogSnapshot([], [first]), [{ ...first, id: 0 }]);
  assertDeepEqual(applyLogSnapshot([{ ...second, id: 0 }], null), []);

  const appended = applyLogAppend([{ ...first, id: 0 }], second, 500);
  assertEqual(appended.length, 2);
  assertDeepEqual(appended[1], { ...second, id: 1 });

  assertDeepEqual(applyLogAppend([{ ...first, id: 0 }], null, 500), [{ ...first, id: 0 }]);

  // 同一秒内的重复行也必须拿到不同的 key，否则虚拟滚动的行高缓存会互相覆盖。
  resetLogIdSequence();
  const duplicated = applyLogSnapshot([], [first, { ...first }]);
  assertEqual(duplicated.length, 2);
  assertEqual(duplicated[0].id === duplicated[1].id, false);
});
