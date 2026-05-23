import { bufferToStoryPainterRows, createMemoryParquetDataset, mapParquetRowToStoryPainterItem } from './parquetDataset';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const rows = [
  { id: '1', nickname: 'A', IMUserId: '1001', time: '10', message: 'one', isDice: false, commandId: '0' },
  { id: 2n, nickname: 'B', IMUserId: '1002', time: 11n, message: 'two', isDice: true, commandId: 5n, commandInfo: '{"ok":true}' },
  { id: 3, nickname: 'C', IMUserId: '1003', time: 12, message: 'three', isDice: false, commandId: 0 },
];

const mapped = mapParquetRowToStoryPainterItem(rows[1]!, 1);
assertEqual(mapped.id, 2);
assertEqual(mapped.commandId, 5);
assertDeepEqual(mapped.commandInfo, { ok: true });

const dataset = createMemoryParquetDataset(rows);
assertEqual(dataset.numRows, 3);
assertDeepEqual((await dataset.readRows(1, 3, ['nickname'])).map(item => [item.id, item.nickname, item.index]), [
  [2, 'B', 1],
  [3, 'C', 2],
]);

const collected: string[] = [];
for await (const chunk of dataset.iterRows({ chunkSize: 2, columns: ['nickname'] })) {
  collected.push(chunk.map(item => item.nickname).join(','));
}
assertDeepEqual(collected, ['A,B', 'C']);

assertDeepEqual(bufferToStoryPainterRows(rows, ['nickname', 'message']).map(item => [item.nickname, item.message]), [
  ['A', 'one'],
  ['B', 'two'],
  ['C', 'three'],
]);
