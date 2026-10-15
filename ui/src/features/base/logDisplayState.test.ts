import type { BaseLogEntry } from './logStream';
import { applyLogDisplayUpdate } from './logDisplayState';
import { expect, it } from 'vitest';

it('日志自动滚动与选择状态转换', async () => {
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

  expect(applyLogDisplayUpdate([], [first, second], true)).toEqual([first, second]);
  expect(applyLogDisplayUpdate([first], [first, second], false)).toEqual([first]);
  expect(applyLogDisplayUpdate([first], [first, second, third], true)).toEqual([
    first,
    second,
    third,
  ]);
});
