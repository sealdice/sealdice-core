import { resolveHashHistoryBase } from './historyBase';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(resolveHashHistoryBase('./'), undefined);
assertEqual(resolveHashHistoryBase(''), undefined);
assertEqual(resolveHashHistoryBase('/fixed/'), '/fixed/');
