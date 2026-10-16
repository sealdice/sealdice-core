import { describe, expect, it } from 'vitest';
import {
  defaultGroupQuitText,
  formatGroupQuitTarget,
  readGroupQuitDefaultText,
  writeGroupQuitDefaultText,
} from './model';

describe('退群模型', () => {
  it('格式化退群目标的名称和账号', () => {
    expect(formatGroupQuitTarget({ groupId: 'group-1', groupName: '测试群', diceId: 'QQ:1' })).toBe(
      'group-1「测试群」，账号 QQ:1'
    );
    expect(formatGroupQuitTarget({ groupId: 'group-2', groupName: '  ' })).toBe('group-2');
  });

  it('持久化默认退群理由并在存储不可用时降级', () => {
    const values = new Map<string, string>();
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => {
        values.set(key, value);
      },
    };
    expect(readGroupQuitDefaultText(storage)).toBe(defaultGroupQuitText);
    writeGroupQuitDefaultText('测试退群文案', storage);
    expect(readGroupQuitDefaultText(storage)).toBe('测试退群文案');
    const broken = {
      getItem(): string | null {
        throw new Error('unavailable');
      },
      setItem(): void {
        throw new Error('unavailable');
      },
    };
    expect(readGroupQuitDefaultText(broken)).toBe(defaultGroupQuitText);
    expect(() => writeGroupQuitDefaultText('不抛出', broken)).not.toThrow();
  });
});
