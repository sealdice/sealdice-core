export type StoryPainterViewMode = 'editor' | 'preview' | 'bbs' | 'bbspineapple' | 'trg';
export type StoryPainterPreviewMode = Exclude<StoryPainterViewMode, 'editor'>;

export function toggleStoryPainterMode(
  current: StoryPainterViewMode,
  requested: StoryPainterPreviewMode,
): StoryPainterViewMode {
  return current === requested ? 'editor' : requested;
}
