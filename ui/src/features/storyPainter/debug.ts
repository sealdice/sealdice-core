export function storyPainterDebug(event: string, payload?: Record<string, unknown>): void {
  if (!import.meta.env.DEV) {
    try {
      if (localStorage.getItem('sd-story-painter-debug') !== '1') return;
    } catch {
      return;
    }
  }
  console.log('[StoryPainter]', event, payload ?? {});
}
