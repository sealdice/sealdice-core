import {
  buildStoryPainterPreviewSources,
  collectMissingVisibleIndexes,
  isStoryPainterVirtualRange,
  virtualItemsToStoryPainterRange,
} from './virtualRange';

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(collectMissingVisibleIndexes({
  visibleIndexes: [10, 11, 12, 13, 14, 15],
  loadedIndexes: new Set([10, 11, 14]),
  range: { start: 2, end: 3 },
  overscan: 1,
}), [12, 13]);

assertDeepEqual(collectMissingVisibleIndexes({
  visibleIndexes: [20, 21, 22],
  loadedIndexes: new Set(),
  range: { start: -10, end: 20 },
  overscan: 2,
}), [20, 21, 22]);

assertDeepEqual(collectMissingVisibleIndexes({
  visibleIndexes: [30, 31, 32, 33],
  loadedIndexes: new Set([30]),
  pendingIndexes: new Set([32]),
  range: { start: 0, end: 3 },
  overscan: 0,
}), [31, 33]);

const loadedItem = {
  id: 1,
  index: 42,
  nickname: 'Alice',
  IMUserId: '1001',
  time: 0,
  message: 'loaded',
  isDice: false,
  commandId: 0,
};

assertDeepEqual(buildStoryPainterPreviewSources([42, 43], [loadedItem]), [
  { key: 42, index: 42 },
  { key: 43, index: 43 },
]);

assertDeepEqual(buildStoryPainterPreviewSources([], [loadedItem]), [
  { key: 42, index: 42, item: loadedItem },
]);

assertDeepEqual([
  isStoryPainterVirtualRange(undefined),
  isStoryPainterVirtualRange({ start: 1 }),
  isStoryPainterVirtualRange({ start: 1, end: 2 }),
], [false, false, true]);

assertDeepEqual(virtualItemsToStoryPainterRange([
  { index: 12, start: 360, size: 30, end: 390, key: 12, lane: 0 },
  { index: 13, start: 390, size: 32, end: 422, key: 13, lane: 0 },
  { index: 14, start: 422, size: 48, end: 470, key: 14, lane: 0 },
]), { start: 12, end: 14 });

assertDeepEqual(virtualItemsToStoryPainterRange([]), undefined);
