export interface StoryPainterVirtualRange {
  start: number;
  end: number;
  padFront?: number;
  padBehind?: number;
}

export interface StoryPainterVirtualItemLike {
  index: number;
  key?: number | string | bigint;
  start?: number;
  end?: number;
  size?: number;
  lane?: number;
}

export function isStoryPainterVirtualRange(value: unknown): value is StoryPainterVirtualRange {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as Partial<StoryPainterVirtualRange>;
  return typeof candidate.start === 'number' && typeof candidate.end === 'number';
}

export function virtualItemsToStoryPainterRange(
  virtualItems: StoryPainterVirtualItemLike[],
): StoryPainterVirtualRange | undefined {
  const first = virtualItems[0];
  const last = virtualItems.at(-1);
  if (!first || !last) return undefined;
  return {
    start: first.index,
    end: last.index,
  };
}

export interface StoryPainterPreviewSource<T> {
  key: number | string;
  index: number;
  item?: T;
}

export function buildStoryPainterPreviewSources<T extends { id?: number; index?: number }>(
  itemIndexes: number[],
  items: T[],
): Array<StoryPainterPreviewSource<T>> {
  if (itemIndexes.length > 0) {
    return itemIndexes.map(index => ({ key: index, index }));
  }
  return items.map((item, index) => ({
    key: item.index ?? item.id ?? index,
    index: item.index ?? index,
    item,
  }));
}

export function collectMissingVisibleIndexes(options: {
  visibleIndexes: number[];
  loadedIndexes: Set<number>;
  pendingIndexes?: Set<number>;
  range: StoryPainterVirtualRange;
  overscan: number;
}): number[] {
  const length = options.visibleIndexes.length;
  if (length === 0) return [];
  const start = Math.max(0, Math.trunc(options.range.start) - options.overscan);
  const end = Math.min(length - 1, Math.trunc(options.range.end) + options.overscan);
  if (end < start) return [];
  const indexes: number[] = [];
  for (let position = start; position <= end; position += 1) {
    const index = options.visibleIndexes[position];
    if (
      index === undefined
      || options.loadedIndexes.has(index)
      || options.pendingIndexes?.has(index)
    ) continue;
    indexes.push(index);
  }
  return indexes;
}
