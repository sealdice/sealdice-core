export const STORY_PAINTER_DEBUG_STORAGE_KEY = 'sd-story-painter-debug';

type DebugStorage = Pick<Storage, 'getItem'>;

type ConsoleLike = Pick<Console, 'log'>;

export interface StoryPainterDebugOptions {
  dev?: boolean;
  storage?: DebugStorage;
  consoleLike?: ConsoleLike;
}

function getBrowserStorage(): DebugStorage | undefined {
  if (typeof localStorage === 'undefined') return undefined;
  return localStorage;
}

export function isStoryPainterDebugEnabled(options: StoryPainterDebugOptions = {}): boolean {
  const dev = options.dev ?? import.meta.env.DEV;
  const storage = options.storage ?? getBrowserStorage();
  if (dev) return true;
  try {
    return storage?.getItem(STORY_PAINTER_DEBUG_STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

export function createStoryPainterDebugLogger(options: StoryPainterDebugOptions = {}) {
  return (event: string, payload?: Record<string, unknown>): void => {
    if (!isStoryPainterDebugEnabled(options)) return;
    const consoleLike = options.consoleLike ?? console;
    consoleLike.log('[StoryPainter]', event, payload ?? {});
  };
}

export const storyPainterDebug = createStoryPainterDebugLogger();
