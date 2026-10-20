import { describe, expect, it } from 'vitest';
import {
  UI_BETA_NOTICE_STORAGE_KEY,
  persistUiBetaNoticeDismissed,
  readUiBetaNoticeDismissed,
} from './notice';

function createStorage(initial: Record<string, string> = {}) {
  const map = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => map.get(key) ?? null,
    setItem: (key: string, value: string) => {
      map.set(key, value);
    },
  };
}

describe('uiBetaNotice', () => {
  it('默认未关闭', () => {
    expect(readUiBetaNoticeDismissed(createStorage())).toBe(false);
  });

  it('关闭后持久化并可读回', () => {
    const storage = createStorage();
    persistUiBetaNoticeDismissed(storage);
    expect(storage.getItem(UI_BETA_NOTICE_STORAGE_KEY)).toBe('1');
    expect(readUiBetaNoticeDismissed(storage)).toBe(true);
  });

  it('存储抛错时安全降级', () => {
    const broken = {
      getItem: () => {
        throw new Error('quota');
      },
      setItem: () => {
        throw new Error('quota');
      },
    };
    expect(readUiBetaNoticeDismissed(broken)).toBe(false);
    expect(() => persistUiBetaNoticeDismissed(broken)).not.toThrow();
  });
});
