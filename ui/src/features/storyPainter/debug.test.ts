import {
  STORY_PAINTER_DEBUG_STORAGE_KEY,
  createStoryPainterDebugLogger,
  isStoryPainterDebugEnabled,
} from './debug';
import { expect, it } from 'vitest';

it('日志阅读器调试参数转换', async () => {
  const createStorage = () => {
    const values = new Map<string, string>();
    return {
      getItem(key: string) {
        return values.get(key) ?? null;
      },
      setItem(key: string, value: string) {
        values.set(key, value);
      },
    };
  };

  const storage = createStorage();
  expect(isStoryPainterDebugEnabled({ dev: false, storage })).toBe(false);
  storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '1');
  expect(isStoryPainterDebugEnabled({ dev: false, storage })).toBe(true);
  storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '0');
  expect(isStoryPainterDebugEnabled({ dev: true, storage })).toBe(true);

  const messages: unknown[][] = [];
  const logger = createStoryPainterDebugLogger({
    dev: false,
    storage,
    consoleLike: {
      log(...args: unknown[]) {
        messages.push(args);
      },
    },
  });

  logger('hidden', { ok: false });
  expect(messages.length).toBe(0);

  storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '1');
  logger('shown', { ok: true });
  expect(messages.length).toBe(1);
  expect(messages[0]?.[0]).toBe('[StoryPainter]');
  expect(messages[0]?.[1]).toBe('shown');
});
