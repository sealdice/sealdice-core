import {
  defaultGroupQuitText,
  readGroupQuitDefaultText,
  writeGroupQuitDefaultText,
} from './quitPreference';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

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
assertEqual(readGroupQuitDefaultText(storage), defaultGroupQuitText);

writeGroupQuitDefaultText('测试退群文案', storage);
assertEqual(readGroupQuitDefaultText(storage), '测试退群文案');

const brokenStorage = {
  getItem() {
    throw new Error('unavailable');
  },
  setItem() {
    throw new Error('unavailable');
  },
};
assertEqual(readGroupQuitDefaultText(brokenStorage), defaultGroupQuitText);
writeGroupQuitDefaultText('不会抛出', brokenStorage);
