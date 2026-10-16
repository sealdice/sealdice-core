import {
  buildStoryPainterPreviewSources,
  collectMissingVisibleIndexes,
  virtualItemsToStoryPainterRange,
} from './virtualRange';
import { expect, it } from 'vitest';

it('虚拟列表范围计算覆盖边界位置', async () => {
  expect(
    collectMissingVisibleIndexes({
      visibleIndexes: [10, 11, 12, 13, 14, 15],
      loadedIndexes: new Set([10, 11, 14]),
      range: { start: 2, end: 3 },
      overscan: 1,
    })
  ).toEqual([12, 13]);

  expect(
    collectMissingVisibleIndexes({
      visibleIndexes: [20, 21, 22],
      loadedIndexes: new Set(),
      range: { start: -10, end: 20 },
      overscan: 2,
    })
  ).toEqual([20, 21, 22]);

  expect(
    collectMissingVisibleIndexes({
      visibleIndexes: [30, 31, 32, 33],
      loadedIndexes: new Set([30]),
      pendingIndexes: new Set([32]),
      range: { start: 0, end: 3 },
      overscan: 0,
    })
  ).toEqual([31, 33]);

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

  expect(buildStoryPainterPreviewSources([42, 43], [loadedItem])).toEqual([
    { key: 42, index: 42 },
    { key: 43, index: 43 },
  ]);

  expect(buildStoryPainterPreviewSources([], [loadedItem])).toEqual([
    { key: 42, index: 42, item: loadedItem },
  ]);

  expect(
    virtualItemsToStoryPainterRange([
      { index: 12, start: 360, size: 30, end: 390, key: 12, lane: 0 },
      { index: 13, start: 390, size: 32, end: 422, key: 13, lane: 0 },
      { index: 14, start: 422, size: 48, end: 470, key: 14, lane: 0 },
    ])
  ).toEqual({ start: 12, end: 14 });

  expect(virtualItemsToStoryPainterRange([])).toEqual(undefined);
});
