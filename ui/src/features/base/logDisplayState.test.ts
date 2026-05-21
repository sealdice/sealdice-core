import type { BaseLogItem } from './logStream';
import { applyLogDisplayUpdate } from './logDisplayState';

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

const third: BaseLogItem = {
  level: 'error',
  msg: 'third',
  ts: 3,
};

assertDeepEqual(applyLogDisplayUpdate([], [first, second], true), [first, second]);
assertDeepEqual(applyLogDisplayUpdate([first], [first, second], false), [first]);
assertDeepEqual(applyLogDisplayUpdate([first], [first, second, third], true), [first, second, third]);
