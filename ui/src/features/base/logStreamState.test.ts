import {
  applyLogAppend,
  applyLogSnapshot,
  applyLogDisplayUpdate,
  type BaseLogEntry,
  type BaseLogItem,
} from './logStreamState';
import { expect, it } from 'vitest';

it('日志快照与追加保持标识唯一和容量限制', async () => {
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

  const snapshot = applyLogSnapshot([], [first]);
  expect(snapshot).toEqual([{ ...first, id: expect.any(Number) }]);
  expect(applyLogSnapshot([{ ...second, id: 0 }], null)).toEqual([]);

  const appended = applyLogAppend(snapshot, second, 500);
  expect(appended.length).toBe(2);
  expect(appended[1]).toMatchObject(second);
  expect(appended[1].id).not.toBe(snapshot[0].id);
  expect(applyLogAppend(appended, second, 2).map(item => item.msg)).toEqual(['second', 'second']);

  expect(applyLogAppend([{ ...first, id: 0 }], null, 500)).toEqual([{ ...first, id: 0 }]);

  // 同一秒内的重复行也必须拿到不同的 key，否则虚拟滚动的行高缓存会互相覆盖。
  const duplicated = applyLogSnapshot([], [first, { ...first }]);
  expect(duplicated.length).toBe(2);
  expect(duplicated[0].id === duplicated[1].id).toBe(false);
});
it('日志展示尊重自动刷新偏好', () => {
  const first: BaseLogEntry = { level: 'info', msg: 'one', ts: 1, id: 0 };
  const second: BaseLogEntry = { level: 'warn', msg: 'two', ts: 2, id: 1 };
  expect(applyLogDisplayUpdate([], [first, second], true)).toEqual([first, second]);
  expect(applyLogDisplayUpdate([first], [first, second], false)).toEqual([first]);
  expect(applyLogDisplayUpdate([first], [first, second], true)).toEqual([first, second]);
});
