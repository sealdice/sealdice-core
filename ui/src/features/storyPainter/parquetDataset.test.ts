import {
  bufferToStoryPainterRows,
  createMemoryParquetDataset,
  mapParquetRowToStoryPainterItem,
} from './parquetDataset';
import { expect, it } from 'vitest';

it('Parquet 数据集索引与记录映射', async () => {
  const rows = [
    {
      id: '1',
      nickname: 'A',
      IMUserId: '1001',
      time: '10',
      message: 'one',
      isDice: false,
      commandId: '0',
    },
    {
      id: 2n,
      nickname: 'B',
      IMUserId: '1002',
      time: 11n,
      message: 'two',
      isDice: true,
      commandId: 5n,
      commandInfo: '{"ok":true}',
    },
    {
      id: 3,
      nickname: 'C',
      IMUserId: '1003',
      time: 12,
      message: 'three',
      isDice: false,
      commandId: 0,
    },
  ];

  const mapped = mapParquetRowToStoryPainterItem(rows[1]!, 1);
  expect(mapped.id).toBe(2);
  expect(mapped.commandId).toBe(5);
  expect(mapped.commandInfo).toEqual({ ok: true });

  const dataset = createMemoryParquetDataset(rows);
  expect(dataset.numRows).toBe(3);
  expect(
    (await dataset.readRows(1, 3, ['nickname'])).map(item => [item.id, item.nickname, item.index])
  ).toEqual([
    [2, 'B', 1],
    [3, 'C', 2],
  ]);

  const collected: string[] = [];
  for await (const chunk of dataset.iterRows({ chunkSize: 2, columns: ['nickname'] })) {
    collected.push(chunk.map(item => item.nickname).join(','));
  }
  expect(collected).toEqual(['A,B', 'C']);

  expect(
    bufferToStoryPainterRows(rows, ['nickname', 'message']).map(item => [
      item.nickname,
      item.message,
    ])
  ).toEqual([
    ['A', 'one'],
    ['B', 'two'],
    ['C', 'three'],
  ]);
});
