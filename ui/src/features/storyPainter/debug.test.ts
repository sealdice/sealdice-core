import {
  STORY_PAINTER_DEBUG_STORAGE_KEY,
  createStoryPainterDebugLogger,
  isStoryPainterDebugEnabled,
} from './debug';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

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
assertEqual(isStoryPainterDebugEnabled({ dev: false, storage }), false);
storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '1');
assertEqual(isStoryPainterDebugEnabled({ dev: false, storage }), true);
storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '0');
assertEqual(isStoryPainterDebugEnabled({ dev: true, storage }), true);

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
assertEqual(messages.length, 0);

storage.setItem(STORY_PAINTER_DEBUG_STORAGE_KEY, '1');
logger('shown', { ok: true });
assertEqual(messages.length, 1);
assertEqual(messages[0]?.[0], '[StoryPainter]');
assertEqual(messages[0]?.[1], 'shown');
