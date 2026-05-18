import type { BaseLogItem } from './logStream';
import { applyLogAppend, applyLogSnapshot } from './logStreamState';

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

assertDeepEqual(applyLogSnapshot([], [first]), [first]);
assertDeepEqual(applyLogSnapshot([second], null), []);

const appended = applyLogAppend([first], second, 500);
assertEqual(appended.length, 2);
assertDeepEqual(appended[1], second);

assertDeepEqual(applyLogAppend([first], null, 500), [first]);
