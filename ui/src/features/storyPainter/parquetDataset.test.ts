// @vitest-environment node
import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';
import { createStoryPainterParquetDataset } from './parquetDataset';

it('使用内置压缩器读取日志 Parquet 文件', async () => {
  const bytes = readFileSync('src/features/storyPainter/testdata/consecutive.parquet');
  const dataset = await createStoryPainterParquetDataset(new Blob([bytes]));

  expect(dataset.numRows).toBeGreaterThan(0);
  const rows = await dataset.readRows(0, Math.min(5, dataset.numRows));
  expect(rows).toHaveLength(Math.min(5, dataset.numRows));
  expect(rows[0]?.message).toBeTypeOf('string');

  // 与日志预览一致，分块读取全部行，覆盖多次 rowStart/rowEnd 查询。
  let count = 0;
  for await (const chunk of dataset.iterRows({ chunkSize: 120 })) {
    count += chunk.length;
  }
  expect(count).toBe(dataset.numRows);
});
