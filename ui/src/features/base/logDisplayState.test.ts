import type { BaseLogEntry } from './logStream';
import { applyLogDisplayUpdate } from './logDisplayState';
import { it } from 'vitest';

it('passes', async () => {
  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
      throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
    }
  };

  const first: BaseLogEntry = {
    level: 'info',
    msg: 'first',
    ts: 1,
    id: 0,
  };

  const second: BaseLogEntry = {
    level: 'warn',
    msg: 'second',
    ts: 2,
    id: 1,
  };

  const third: BaseLogEntry = {
    level: 'error',
    msg: 'third',
    ts: 3,
    id: 2,
  };

  assertDeepEqual(applyLogDisplayUpdate([], [first, second], true), [first, second]);
  assertDeepEqual(applyLogDisplayUpdate([first], [first, second], false), [first]);
  assertDeepEqual(applyLogDisplayUpdate([first], [first, second, third], true), [
    first,
    second,
    third,
  ]);
});
