import type { StoryPainterLogItem } from './types';
import { createStoryPainterParquetDataset } from './parquetDataset';

export async function readStoryPainterParquet(blob: Blob): Promise<StoryPainterLogItem[]> {
  const dataset = await createStoryPainterParquetDataset(blob);
  return await dataset.readAll();
}
