import { computed, reactive, shallowRef, watch } from 'vue';
import randomColor from 'randomcolor';
import {
  buildStoryPainterChars,
  buildStoryPainterPcMap,
  buildStoryPainterPreviewItems,
  deleteStoryPainterChar,
  renameStoryPainterChar,
  storyPainterPalette,
} from './state';
import {
  defaultStoryPainterOptions,
  packStoryPainterNameId,
  type StoryPainterChar,
  type StoryPainterLogItem,
  type StoryPainterOptions,
  type StoryPainterPreviewDisplayMode,
} from './types';
import type { StoryPainterParquetDataset } from './parquetDataset';
import { storyPainterDebug } from './debug';
import { normalizeStoryPainterMessage } from './formatters';
import { normalizeStoryPainterPlainMessage } from './renderers';

const storageKey = 'storyPainterPcNameColorMap';
const indexColumns = ['id', 'nickname', 'IMUserId', 'time', 'isDice'] as const;
const previewReadChunkSize = 120;
const rowCacheLimit = 3000;

export function useStoryPainter() {
  const items = shallowRef<StoryPainterLogItem[]>([]);
  const datasetPreviewItems = shallowRef<StoryPainterLogItem[]>([]);
  const dataset = shallowRef<StoryPainterParquetDataset | null>(null);
  const fullItemsLoaded = shallowRef(false);
  const chars = shallowRef<StoryPainterChar[]>([]);
  const swatches = shallowRef<string[]>(randomColor({ count: 16 }) as string[]);
  const mode = shallowRef<'editor' | StoryPainterPreviewDisplayMode>('editor');
  const editorHighlight = shallowRef(false);
  const trgIsAddVoiceMark = shallowRef(false);
  const forumOptions = reactive({
    bbsUseSpaceWithMultiLine: false,
    bbsUseColorName: false,
  });
  const exportOptions = reactive<StoryPainterOptions>(defaultStoryPainterOptions());
  const pcNameColorMap = loadColorMap();
  const rowCache = new Map<number, StoryPainterLogItem>();
  const rowChunkInflight = new Map<number, Promise<void>>();
  let rowSourceVersion = 0;
  const charRenames: Array<{ char: StoryPainterChar; name: string }> = [];
  const deletedRowIndexes = new Set<number>();
  let previewVersion = 0;

  const pcMap = computed(() => buildStoryPainterPcMap(chars.value));
  const itemsByIndex = computed(() => new Map(items.value.map(item => [item.index!, item])));
  const previewOptions = computed(() => ({
    ...exportOptions,
    imageHide: exportOptions.imageHide || mode.value === 'bbs' || mode.value === 'bbspineapple',
  }));
  const previewItems = computed(() =>
    fullItemsLoaded.value
      ? filterPreviewItems(items.value, chars.value, previewOptions.value)
      : datasetPreviewItems.value
  );

  watch([items, chars, previewOptions, mode], () => {
    if (!fullItemsLoaded.value && mode.value !== 'editor') return refreshPreview();
  });

  function filterPreviewItems(
    rows: StoryPainterLogItem[],
    currentChars: StoryPainterChar[],
    options: StoryPainterOptions,
    pineapple = mode.value === 'bbspineapple'
  ): StoryPainterLogItem[] {
    return buildStoryPainterPreviewItems(rows, currentChars, item =>
      pineapple
        ? normalizeStoryPainterPlainMessage(item, currentChars, options)
        : normalizeStoryPainterMessage(item, currentChars, options, false)
    );
  }

  function resetReadState(): void {
    rowSourceVersion += 1;
    rowCache.clear();
    rowChunkInflight.clear();
    charRenames.length = 0;
    deletedRowIndexes.clear();
    datasetPreviewItems.value = [];
  }

  function setAllItemsFromEditor(nextItems: StoryPainterLogItem[]): void {
    dataset.value = null;
    fullItemsLoaded.value = true;
    resetReadState();
    items.value = nextItems.map((item, index) => ({ ...item, index }));
    chars.value = syncChars(chars.value, buildStoryPainterChars(nextItems, pcNameColorMap));
  }

  async function setDataset(nextDataset: StoryPainterParquetDataset): Promise<void> {
    dataset.value = nextDataset;
    fullItemsLoaded.value = false;
    resetReadState();
    const version = rowSourceVersion;
    const indexRows = await nextDataset.readRows(0, nextDataset.numRows, [...indexColumns]);
    if (version !== rowSourceVersion) return;
    items.value = indexRows;
    chars.value = buildStoryPainterChars(indexRows, pcNameColorMap);
    await refreshPreview();
  }

  async function refreshPreview(): Promise<void> {
    const version = ++previewVersion;
    const sourceVersion = rowSourceVersion;
    if (fullItemsLoaded.value || !dataset.value) return;
    const metadata: StoryPainterLogItem[] = [];
    for await (const chunk of iterPreviewChunks()) {
      if (version !== previewVersion || sourceVersion !== rowSourceVersion) return;
      chunk.forEach(item => metadata.push({ ...item, message: '', commandInfo: undefined }));
    }
    if (version === previewVersion && sourceVersion === rowSourceVersion)
      datasetPreviewItems.value = metadata;
  }

  function updateChar(index: number, patch: Partial<StoryPainterChar>): void {
    const next = [...chars.value];
    const current = next[index];
    if (!current) return;
    const updated = { ...current, ...patch };
    next[index] = updated;
    chars.value = next;
    if (patch.color) {
      pcNameColorMap.set(updated.name, patch.color);
      saveColorMap(pcNameColorMap);
    }
  }

  function renameChar(index: number, nextName: string): void {
    const current = chars.value[index];
    if (!current || !nextName.trim() || current.name === nextName) return;
    const trimmedName = nextName.trim();
    items.value = renameStoryPainterChar(items.value, current, trimmedName);
    if (!fullItemsLoaded.value) charRenames.push({ char: current, name: trimmedName });
    updateChar(index, { name: trimmedName });
  }

  function deleteChar(index: number): void {
    const current = chars.value[index];
    if (!current) return;
    if (!fullItemsLoaded.value) {
      items.value.forEach(item => {
        if (item.IMUserId === current.IMUserId && item.nickname === current.name)
          deletedRowIndexes.add(item.index!);
      });
    }
    items.value = deleteStoryPainterChar(items.value, current);
    chars.value = chars.value.filter((_, itemIndex) => itemIndex !== index);
  }

  function refreshSwatches(): void {
    swatches.value = randomColor({ count: 16 }) as string[];
  }

  function colorByItem(item: StoryPainterLogItem): string {
    return pcMap.value.get(packStoryPainterNameId(item))?.color || '#4b5563';
  }

  async function readPreviewItems(indexes: number[]): Promise<StoryPainterLogItem[]> {
    const uniqueIndexes = [...new Set(indexes)].filter(index => Number.isFinite(index));
    if (uniqueIndexes.length === 0) return [];
    const source = dataset.value;
    if (fullItemsLoaded.value || !source) {
      return indexes
        .map(index => itemsByIndex.value.get(index))
        .filter((item): item is StoryPainterLogItem => Boolean(item));
    }
    const missing = uniqueIndexes.filter(index => !rowCache.has(index));
    const chunkStarts = [
      ...new Set(
        missing.map(index => Math.floor(index / previewReadChunkSize) * previewReadChunkSize)
      ),
    ];
    const sourceVersion = rowSourceVersion;
    storyPainterDebug('painter:read-preview-items', {
      requestedCount: indexes.length,
      uniqueCount: uniqueIndexes.length,
      cacheHitCount: uniqueIndexes.length - missing.length,
      missingCount: missing.length,
      firstRequestedIndex: indexes[0],
      lastRequestedIndex: indexes.at(-1),
      chunkStarts,
      cacheSize: rowCache.size,
    });
    await Promise.all(chunkStarts.map(start => readPreviewChunk(source, start, sourceVersion)));
    if (sourceVersion !== rowSourceVersion) return [];
    const rows = applyEdits(
      indexes
        .map(index => rowCache.get(index))
        .filter((item): item is StoryPainterLogItem => Boolean(item))
    );
    storyPainterDebug('painter:read-preview-items-done', {
      requestedCount: indexes.length,
      returnedCount: rows.length,
      firstReturnedIndex: rows[0]?.index,
      lastReturnedIndex: rows.at(-1)?.index,
      cacheSize: rowCache.size,
    });
    return rows;
  }

  async function ensureFullItems(): Promise<void> {
    const source = dataset.value;
    if (!source) {
      fullItemsLoaded.value = true;
      return;
    }
    if (fullItemsLoaded.value) return;
    const version = rowSourceVersion;
    const rows = await source.readRows(0, source.numRows);
    if (version !== rowSourceVersion) return;
    items.value = applyEdits(rows);
    fullItemsLoaded.value = true;
    resetReadState();
  }

  async function* iterPreviewChunks(chunkSize = 2000): AsyncGenerator<StoryPainterLogItem[]> {
    const source = dataset.value;
    if (fullItemsLoaded.value || !source) {
      yield previewItems.value;
      return;
    }
    const currentChars = chars.value;
    const options = previewOptions.value;
    const pineapple = mode.value === 'bbspineapple';
    const renames = [...charRenames];
    const deleted = new Set(deletedRowIndexes);
    for await (const chunk of source.iterRows({ chunkSize })) {
      yield filterPreviewItems(
        applyEdits(chunk, renames, deleted),
        currentChars,
        options,
        pineapple
      );
    }
  }

  function applyEdits(
    rows: StoryPainterLogItem[],
    renames = charRenames,
    deleted = deletedRowIndexes
  ): StoryPainterLogItem[] {
    let next = rows.filter(item => !deleted.has(item.index!));
    for (const { char, name } of renames) {
      next = renameStoryPainterChar(next, char, name);
    }
    return next;
  }

  function setCachedRow(index: number, item: StoryPainterLogItem): void {
    if (rowCache.has(index)) rowCache.delete(index);
    rowCache.set(index, item);
    while (rowCache.size > rowCacheLimit) {
      const oldest = rowCache.keys().next().value as number | undefined;
      if (oldest === undefined) break;
      rowCache.delete(oldest);
    }
  }

  function readPreviewChunk(
    source: StoryPainterParquetDataset,
    start: number,
    sourceVersion: number
  ): Promise<void> {
    const inflight = rowChunkInflight.get(start);
    if (inflight) {
      storyPainterDebug('painter:read-preview-chunk-reuse', {
        start,
        end: Math.min(start + previewReadChunkSize, source.numRows),
      });
      return inflight;
    }
    const promise = (async () => {
      const end = Math.min(start + previewReadChunkSize, source.numRows);
      storyPainterDebug('painter:read-preview-chunk-start', {
        start,
        end,
        sourceVersion,
        currentSourceVersion: rowSourceVersion,
      });
      const rows = await source.readRows(start, end);
      if (sourceVersion !== rowSourceVersion) {
        storyPainterDebug('painter:read-preview-chunk-skip', {
          start,
          end,
          reason: 'stale-source-version',
          sourceVersion,
          currentSourceVersion: rowSourceVersion,
          rowCount: rows.length,
        });
        return;
      }
      let cachedCount = 0;
      rows.forEach(item => {
        if (item.index !== undefined) {
          setCachedRow(item.index, item);
          cachedCount += 1;
        }
      });
      storyPainterDebug('painter:read-preview-chunk-done', {
        start,
        end,
        rowCount: rows.length,
        cachedCount,
        cacheSize: rowCache.size,
        firstRowIndex: rows[0]?.index,
        lastRowIndex: rows.at(-1)?.index,
      });
    })().finally(() => {
      if (rowChunkInflight.get(start) === promise) rowChunkInflight.delete(start);
    });
    rowChunkInflight.set(start, promise);
    return promise;
  }

  return {
    items,
    previewItems,
    dataset,
    fullItemsLoaded,
    chars,
    swatches,
    mode,
    editorHighlight,
    trgIsAddVoiceMark,
    forumOptions,
    exportOptions,
    setDataset,
    setAllItemsFromEditor,
    readPreviewItems,
    ensureFullItems,
    iterPreviewChunks,
    updateChar,
    renameChar,
    deleteChar,
    refreshSwatches,
    colorByItem,
  };
}

function syncChars(current: StoryPainterChar[], next: StoryPainterChar[]): StoryPainterChar[] {
  const currentMap = new Map(current.map(item => [packStoryPainterNameId(item), item]));
  return next.map((item, index) => ({
    ...item,
    color:
      currentMap.get(packStoryPainterNameId(item))?.color ||
      item.color ||
      storyPainterPalette[index % storyPainterPalette.length],
    role: currentMap.get(packStoryPainterNameId(item))?.role || item.role,
  }));
}

function loadColorMap(): Map<string, string> {
  try {
    return new Map(JSON.parse(localStorage.getItem(storageKey) || '[]'));
  } catch {
    return new Map();
  }
}

function saveColorMap(value: Map<string, string>): void {
  try {
    localStorage.setItem(storageKey, JSON.stringify([...value]));
  } catch {
    // 忽略本地存储异常，颜色仍在当前会话生效。
  }
}
