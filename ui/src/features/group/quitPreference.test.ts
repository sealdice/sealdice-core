import {
  defaultGroupQuitText,
  readGroupQuitDefaultText,
  writeGroupQuitDefaultText,
} from './quitPreference';
import { expect, it } from 'vitest';

it('退出群组偏好在状态间保留', async () => {
  function createStorage() {
    const values = new Map<string, string>();
    return {
      getItem(key: string) {
        return values.get(key) ?? null;
      },
      setItem(key: string, value: string) {
        values.set(key, value);
      },
    };
  }

  const storage = createStorage();
  expect(readGroupQuitDefaultText(storage)).toBe(defaultGroupQuitText);

  writeGroupQuitDefaultText('测试退群文案', storage);
  expect(readGroupQuitDefaultText(storage)).toBe('测试退群文案');

  const brokenStorage = {
    getItem() {
      throw new Error('unavailable');
    },
    setItem() {
      throw new Error('unavailable');
    },
  };
  expect(readGroupQuitDefaultText(brokenStorage)).toBe(defaultGroupQuitText);
  writeGroupQuitDefaultText('不会抛出', brokenStorage);
});
