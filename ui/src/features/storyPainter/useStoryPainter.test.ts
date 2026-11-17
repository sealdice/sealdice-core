import { parquetMetadataAsync, parquetReadObjects, type FileMetaData } from 'hyparquet';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { effectScope, nextTick } from 'vue';
import { createStoryPainterParquetDataset, type StoryPainterParquetRow } from './parquetDataset';
import { useStoryPainter } from './useStoryPainter';

vi.mock('hyparquet', () => ({
  parquetMetadataAsync: vi.fn(),
  parquetReadObjects: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

const scopes: ReturnType<typeof effectScope>[] = [];
function createPainter() {
  const scope = effectScope();
  scopes.push(scope);
  return scope.run(useStoryPainter)!;
}
afterEach(() => scopes.splice(0).forEach(scope => scope.stop()));

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
  const painter = createPainter();
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
  const visible: string[][] = [];
  for await (const chunk of painter.iterPreviewChunks()) {
    visible.push(...chunk.map(item => [item.nickname, item.message]));
  }
  expect(visible).toEqual([['Seal', '<Alicia> roll']]);
});

it('预览批量读取复用并发请求和缓存，并按请求顺序返回跨块记录', async () => {
  const painter = createPainter();
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

it('Parquet 缓存、分块导出和全文编辑沿用连续改名，含连字符账号及名字重复使用', async () => {
  const painter = createPainter();
  await painter.setDataset(await createDataset([{ ...alice, IMUserId: 'QQ-1001' }, seal]));
  await painter.readPreviewItems([0, 1]);
  for (const name of ['Alicia', 'Another', 'Alice', 'Final']) painter.renameChar(0, name);
  const rows = await painter.readPreviewItems([0, 1]);
  expect(rows.map(item => [item.nickname, item.message])).toEqual([
    ['Final', 'one'],
    ['Seal', '<Final> roll'],
  ]);
  const exported = [];
  for await (const chunk of painter.iterPreviewChunks(1)) exported.push(...chunk);
  expect(exported).toEqual(rows);
  painter.deleteChar(0);
  expect((await painter.readPreviewItems([0, 1])).map(item => item.nickname)).toEqual(['Seal']);
  await painter.ensureFullItems();
  expect(painter.items.value.map(item => item.message)).toEqual(['<Final> roll']);
  painter.setAllItemsFromEditor([
    { ...painter.items.value[0]!, message: 'edited' },
    { ...alice, message: 'new row' },
  ]);
  expect((await painter.readPreviewItems([0, 1])).map(item => item.message)).toEqual([
    'edited',
    'new row',
  ]);
  const edited = [];
  for await (const chunk of painter.iterPreviewChunks()) edited.push(...chunk);
  expect(edited.map(item => item.message)).toEqual(['edited', 'new row']);
});

it('预览随过滤条件和角色更新，多行论坛跳过仅图片发言，导出顺序一致', async () => {
  const painter = createPainter();
  await painter.setDataset(
    await createDataset([
      alice,
      { ...alice, id: 2, message: '.ra' },
      { ...alice, id: 3, message: '(场外)' },
      { ...alice, id: 4, message: '[CQ:image,file=https://example.com/a.png]' },
      seal,
    ])
  );
  painter.mode.value = 'preview';
  painter.exportOptions.commandHide = true;
  painter.exportOptions.offTopicHide = true;
  await vi.waitFor(() =>
    expect(painter.previewItems.value.map(item => item.index)).toEqual([0, 3, 4])
  );
  painter.mode.value = 'bbspineapple';
  await vi.waitFor(() =>
    expect(painter.previewItems.value.map(item => item.index)).toEqual([0, 4])
  );
  painter.updateChar(1, { role: '隐藏' });
  await vi.waitFor(() => expect(painter.previewItems.value.map(item => item.index)).toEqual([0]));
  painter.exportOptions.commandHide = false;
  painter.exportOptions.offTopicHide = false;
  await vi.waitFor(() =>
    expect(painter.previewItems.value.map(item => item.index)).toEqual([0, 1, 2])
  );
  const exported = [];
  for await (const chunk of painter.iterPreviewChunks(2)) exported.push(...chunk);
  expect(
    await painter.readPreviewItems(painter.previewItems.value.map(item => item.index!))
  ).toEqual(exported);
});

it('较早的过滤扫描晚完成时不能覆盖新的过滤结果', async () => {
  const painter = createPainter();
  const dataset = await createDataset([alice, { ...alice, id: 2, message: '.ra' }]);
  await painter.setDataset(dataset);
  let release!: () => void;
  const gate = new Promise<void>(resolve => {
    release = resolve;
  });
  let finish!: () => void;
  const finished = new Promise<void>(resolve => {
    finish = resolve;
  });
  const read = dataset.readRows.bind(dataset);
  vi.spyOn(dataset, 'iterRows').mockImplementationOnce(async function* () {
    try {
      await gate;
      yield await read(0, dataset.numRows);
    } finally {
      finish();
    }
  });
  painter.mode.value = 'preview';
  await nextTick();
  painter.exportOptions.commandHide = true;
  await vi.waitFor(() => expect(painter.previewItems.value.map(item => item.index)).toEqual([0]));
  release();
  await finished;
  expect(painter.previewItems.value.map(item => item.index)).toEqual([0]);
});

it('切换日志后迟到的正文和全文读取不能替换新日志', async () => {
  const painter = createPainter();
  const original = await createDataset([alice]);
  await painter.setDataset(original);
  let release!: () => void;
  const gate = new Promise<void>(resolve => {
    release = resolve;
  });
  vi.spyOn(original, 'readRows').mockImplementation(async () => {
    await gate;
    return [{ ...alice, index: 0 }];
  });
  const oldPreview = painter.readPreviewItems([0]);
  const oldEditor = painter.ensureFullItems();
  await painter.setDataset(await createDataset([{ ...alice, message: 'new log' }]));
  release();
  expect(await oldPreview).toEqual([]);
  await oldEditor;
  expect(painter.fullItemsLoaded.value).toBe(false);
  expect((await painter.readPreviewItems([0]))[0]?.message).toBe('new log');
});
