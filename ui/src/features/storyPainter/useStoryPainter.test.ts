import { parquetMetadataAsync, parquetReadObjects, type FileMetaData } from 'hyparquet';
import { beforeEach, expect, it, vi } from 'vitest';
import { createStoryPainterParquetDataset, type StoryPainterParquetRow } from './parquetDataset';
import { useStoryPainter } from './useStoryPainter';

vi.mock('hyparquet', () => ({
  parquetMetadataAsync: vi.fn(),
  parquetReadObjects: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

async function createDataset(rows: StoryPainterParquetRow[]) {
  vi.mocked(parquetMetadataAsync).mockResolvedValue({
    num_rows: BigInt(rows.length),
  } as FileMetaData);
  vi.mocked(parquetReadObjects).mockImplementation(async options =>
    rows.slice(options.rowStart, options.rowEnd)
  );
  return createStoryPainterParquetDataset(new Blob());
}

const alice = {
  id: 1,
  nickname: 'Alice',
  IMUserId: '1001',
  time: 10,
  message: 'one',
  isDice: false,
  commandId: 0,
};
const seal = {
  ...alice,
  id: 2,
  nickname: 'Seal',
  IMUserId: 'bot',
  message: '<Alice> roll',
  isDice: true,
  commandId: 5n,
  commandInfo: '{"ok":true}',
};

it('Parquet 按范围和列读取，保留原始行索引与指令数据', async () => {
  const dataset = await createDataset([alice, seal]);
  expect(dataset.numRows).toBe(2);
  const rows = await dataset.readRows(1, 20);
  expect(rows[0]).toMatchObject({ index: 1, commandId: 5, commandInfo: { ok: true } });
  expect(await dataset.readRows(2, 1)).toEqual([]);
  expect((await dataset.readRows(-10, 1, ['nickname']))[0]).toMatchObject({
    index: 0,
    nickname: 'Alice',
    message: '',
  });

  const collected: string[][] = [];
  for await (const chunk of dataset.iterRows({ chunkSize: 1, columns: ['nickname'] })) {
    collected.push(chunk.map(item => item.nickname));
  }
  expect(collected).toEqual([['Alice'], ['Seal']]);
});

it('编辑器按需加载全文，预览与导出沿用角色改名和删除', async () => {
  const painter = useStoryPainter();
  await painter.setDataset(await createDataset([alice, seal]));
  expect(painter.fullItemsLoaded.value).toBe(false);
  expect(painter.items.value.map(item => item.message)).toEqual(['', '']);
  expect((await painter.readPreviewItems([0]))[0]?.message).toBe('one');

  await painter.ensureFullItems();
  expect(painter.fullItemsLoaded.value).toBe(true);
  expect(painter.items.value.map(item => item.message)).toEqual(['one', '<Alice> roll']);

  painter.renameChar(0, 'Alicia');
  expect(
    (await painter.readPreviewItems([0, 1])).map(item => [item.nickname, item.message])
  ).toEqual([
    ['Alicia', 'one'],
    ['Seal', '<Alicia> roll'],
  ]);
  painter.deleteChar(0);
  await painter.refreshPreview();
  const visible: string[][] = [];
  for await (const chunk of painter.iterPreviewChunks()) {
    visible.push(...chunk.map(item => [item.nickname, item.message]));
  }
  expect(visible).toEqual([['Seal', '<Alicia> roll']]);
});

it('预览批量读取复用并发请求和缓存，并按请求顺序返回跨块记录', async () => {
  const painter = useStoryPainter();
  await painter.setDataset(
    await createDataset(
      Array.from({ length: 260 }, (_, index) => ({
        ...alice,
        id: index + 1,
        message: `message-${index}`,
      }))
    )
  );
  vi.mocked(parquetReadObjects).mockClear();

  const [rows] = await Promise.all([
    painter.readPreviewItems([124, 121, 123]),
    painter.readPreviewItems([124]),
  ]);
  expect(rows.map(item => item.message)).toEqual(['message-124', 'message-121', 'message-123']);
  expect(parquetReadObjects).toHaveBeenCalledTimes(1);

  expect((await painter.readPreviewItems([123, 124])).map(item => item.message)).toEqual([
    'message-123',
    'message-124',
  ]);
  expect(parquetReadObjects).toHaveBeenCalledTimes(1);
  expect((await painter.readPreviewItems([250, 121])).map(item => item.message)).toEqual([
    'message-250',
    'message-121',
  ]);
  expect(parquetReadObjects).toHaveBeenCalledTimes(2);
});
