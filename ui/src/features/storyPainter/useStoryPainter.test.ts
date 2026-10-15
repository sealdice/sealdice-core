import { createMemoryParquetDataset } from './parquetDataset';
import type {
  StoryPainterParquetColumn,
  StoryPainterParquetDataset,
  StoryPainterParquetRow,
} from './parquetDataset';
import type { StoryPainterLogItem } from './types';
import { useStoryPainter } from './useStoryPainter';
import { expect, it } from 'vitest';

it('日志阅读模型随数据和选择更新', async () => {
  const painter = useStoryPainter();
  await painter.setDataset(
    createMemoryParquetDataset([
      {
        id: 1,
        nickname: 'Alice',
        IMUserId: '1001',
        time: 10,
        message: 'one',
        isDice: false,
        commandId: 0,
      },
      {
        id: 2,
        nickname: 'Bob',
        IMUserId: '1002',
        time: 11,
        message: 'two',
        isDice: false,
        commandId: 0,
      },
    ])
  );

  expect(painter.fullItemsLoaded.value).toBe(false);
  expect(painter.items.value.map(item => item.message)).toEqual(['', '']);
  expect((await painter.readPreviewItem(0))?.message).toBe('one');

  await painter.ensureFullItems();
  expect(painter.fullItemsLoaded.value).toBe(true);
  expect(painter.items.value.map(item => item.message)).toEqual(['one', 'two']);

  const datasetPainter = useStoryPainter();
  await datasetPainter.setDataset(
    createMemoryParquetDataset([
      {
        id: 1,
        nickname: 'Alice',
        IMUserId: '1001',
        time: 10,
        message: 'one',
        isDice: false,
        commandId: 0,
      },
      {
        id: 2,
        nickname: 'Seal',
        IMUserId: 'bot',
        time: 11,
        message: '<Alice> roll',
        isDice: true,
        commandId: 1,
      },
    ])
  );

  datasetPainter.renameChar(0, 'Alicia');
  expect((await datasetPainter.readPreviewItem(0))?.nickname).toBe('Alicia');
  expect((await datasetPainter.readPreviewItem(1))?.message).toBe('<Alicia> roll');

  datasetPainter.deleteChar(0);
  await datasetPainter.refreshPreview();
  const visible: Array<[string, string]> = [];
  for await (const chunk of datasetPainter.iterPreviewChunks()) {
    visible.push(...chunk.map(item => [item.nickname, item.message] as [string, string]));
  }
  expect(visible).toEqual([['Seal', '<Alicia> roll']]);

  const manyRows: StoryPainterParquetRow[] = Array.from({ length: 260 }, (_, index) => ({
    id: index + 1,
    nickname: `P${index % 3}`,
    IMUserId: `u${index % 3}`,
    time: index,
    message: `message-${index}`,
    isDice: false,
    commandId: 0,
  }));
  const counted = createCountingDataset(manyRows);
  const batchPainter = useStoryPainter();
  await batchPainter.setDataset(counted.dataset);
  const rows = await batchPainter.readPreviewItems([124, 121, 123]);
  expect(rows.map(item => item.message)).toEqual(['message-124', 'message-121', 'message-123']);
  expect(
    counted.calls
      .filter(call => call.columns.includes('message'))
      .map(call => [call.start, call.end])
  ).toEqual([[120, 240]]);

  const far = await batchPainter.readPreviewItem(250);
  expect(far?.message).toBe('message-250');

  const concurrentCounted = createCountingDataset(manyRows);
  const concurrentPainter = useStoryPainter();
  await concurrentPainter.setDataset(concurrentCounted.dataset);
  await Promise.all([
    concurrentPainter.readPreviewItems([121]),
    concurrentPainter.readPreviewItems([124]),
  ]);
  expect(
    concurrentCounted.calls
      .filter(call => call.columns.includes('message'))
      .map(call => [call.start, call.end])
  ).toEqual([[120, 240]]);

  function createCountingDataset(rows: StoryPainterParquetRow[]): {
    dataset: StoryPainterParquetDataset;
    calls: Array<{ start: number; end: number; columns: string[] }>;
  } {
    const base = createMemoryParquetDataset(rows);
    const calls: Array<{ start: number; end: number; columns: string[] }> = [];
    const dataset: StoryPainterParquetDataset = {
      numRows: base.numRows,
      async readRows(
        start: number,
        end: number,
        columns?: StoryPainterParquetColumn[]
      ): Promise<StoryPainterLogItem[]> {
        calls.push({ start, end, columns: columns ?? [] });
        return await base.readRows(start, end, columns);
      },
      iterRows(options) {
        return base.iterRows(options);
      },
      readAll(columns?: StoryPainterParquetColumn[]) {
        return base.readAll(columns);
      },
    };
    return { dataset, calls };
  }
});
