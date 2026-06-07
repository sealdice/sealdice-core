import { computed, reactive, shallowRef } from 'vue';
import randomColor from 'randomcolor';
import {
  buildStoryPainterChars,
  buildStoryPainterPcMap,
  buildStoryPainterPreviewItems,
  deleteStoryPainterChar,
  isStoryPainterHidden,
  renameStoryPainterChar,
  storyPainterPalette,
} from './state';
import {
  defaultStoryPainterOptions,
  packStoryPainterNameId,
  type StoryPainterChar,
  type StoryPainterLogItem,
  type StoryPainterOptions,
} from './types';
import type { StoryPainterParquetDataset } from './parquetDataset';
import { storyPainterDebug } from './debug';
import { normalizeStoryPainterMessage } from './formatters';
import { replaceAllText } from './string';
import type { StoryPainterViewMode } from './viewMode';

const storageKey = 'storyPainterPcNameColorMap';
const indexColumns = ['id', 'nickname', 'IMUserId', 'time', 'isDice'] as const;
const previewColumns = ['id', 'nickname', 'IMUserId', 'time', 'message', 'isDice', 'commandId', 'commandInfo', 'uniformId'] as const;
const previewReadChunkSize = 120;
const rowCacheLimit = 3000;

export function useStoryPainter() {
  const items = shallowRef<StoryPainterLogItem[]>([]);
  const previewItems = shallowRef<StoryPainterLogItem[]>([]);
  const dataset = shallowRef<StoryPainterParquetDataset | null>(null);
  const fullItemsLoaded = shallowRef(false);
  const visibleIndexes = shallowRef<number[]>([]);
  const chars = shallowRef<StoryPainterChar[]>([]);
  const swatches = shallowRef<string[]>(randomColor({ count: 16 }) as string[]);
  const mode = shallowRef<StoryPainterViewMode>('editor');
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
  const charNameOverrides = new Map<string, string>();
  const deletedOriginalCharKeys = new Set<string>();

  const pcMap = computed(() => buildStoryPainterPcMap(chars.value));
  const previewCount = computed(() => visibleIndexes.value.length || previewItems.value.length);

  function setItems(nextItems: StoryPainterLogItem[]): void {
    dataset.value = null;
    fullItemsLoaded.value = true;
    rowSourceVersion += 1;
    rowCache.clear();
    rowChunkInflight.clear();
    items.value = nextItems.map((item) => ({ ...item, message: replaceAllText(item.message, '\r\n', '\n') }));
    chars.value = buildStoryPainterChars(items.value, pcNameColorMap);
    refreshPreviewFromItems();
  }

  function setAllItemsFromEditor(nextItems: StoryPainterLogItem[]): void {
    dataset.value = null;
    fullItemsLoaded.value = true;
    rowSourceVersion += 1;
    rowCache.clear();
    rowChunkInflight.clear();
    items.value = nextItems;
    chars.value = syncChars(chars.value, buildStoryPainterChars(nextItems, pcNameColorMap));
    refreshPreviewFromItems();
  }

  async function setDataset(nextDataset: StoryPainterParquetDataset): Promise<void> {
    dataset.value = nextDataset;
    fullItemsLoaded.value = false;
    rowSourceVersion += 1;
    const indexRows = await nextDataset.readRows(0, nextDataset.numRows, [...indexColumns]);
    items.value = indexRows;
    chars.value = buildStoryPainterChars(indexRows, pcNameColorMap);
    rowCache.clear();
    rowChunkInflight.clear();
    charNameOverrides.clear();
    deletedOriginalCharKeys.clear();
    await refreshPreviewFromDataset();
  }

  async function refreshPreview(): Promise<void> {
    if (dataset.value) {
      await refreshPreviewFromDataset();
      return;
    }
    refreshPreviewFromItems();
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
    const overrideKey = packStoryPainterNameId(current);
    items.value = renameStoryPainterChar(items.value, current, trimmedName);
    previewItems.value = renameStoryPainterChar(previewItems.value, current, trimmedName);
    for (const [rowIndex, row] of rowCache) {
      rowCache.set(rowIndex, renameStoryPainterChar([row], current, trimmedName)[0] ?? row);
    }
    charNameOverrides.set(overrideKey, trimmedName);
    updateChar(index, { name: trimmedName });
  }

  function deleteChar(index: number): void {
    const current = chars.value[index];
    if (!current) return;
    deletedOriginalCharKeys.add(originalCharKey(current));
    const removed = new Set(items.value
      .filter(item => item.IMUserId === current.IMUserId && item.nickname === current.name)
      .map(item => item.index ?? 0));
    items.value = deleteStoryPainterChar(items.value, current);
    previewItems.value = deleteStoryPainterChar(previewItems.value, current);
    visibleIndexes.value = visibleIndexes.value.filter(itemIndex => !removed.has(itemIndex));
    chars.value = chars.value.filter((_, itemIndex) => itemIndex !== index);
  }

  function refreshSwatches(): void {
    swatches.value = randomColor({ count: 16 }) as string[];
  }

  function colorByItem(item: StoryPainterLogItem): string {
    return pcMap.value.get(packStoryPainterNameId(item))?.color || '#4b5563';
  }

  function hidden(item: StoryPainterLogItem): boolean {
    return isStoryPainterHidden(item, chars.value);
  }

  function refreshPreviewFromItems(): void {
    previewItems.value = buildStoryPainterPreviewItems(items.value, chars.value, exportOptions, (item) =>
      normalizeStoryPainterMessage(item, chars.value, exportOptions, false),
    );
    visibleIndexes.value = previewItems.value.map(item => item.index ?? item.id - 1);
  }

  async function refreshPreviewFromDataset(): Promise<void> {
    const source = dataset.value;
    if (!source) return;
    const indexes: number[] = [];
    for await (const chunk of source.iterRows({ columns: [...previewColumns], chunkSize: 2000 })) {
      const overridden = chunk.filter(item => !isDeletedOriginalItem(item)).map(applyCharOverrides);
      const visible = buildStoryPainterPreviewItems(overridden, chars.value, exportOptions, (item) =>
        normalizeStoryPainterMessage(item, chars.value, exportOptions, false),
      );
      visible.forEach((item) => {
        indexes.push(item.index ?? 0);
      });
    }
    visibleIndexes.value = indexes;
    previewItems.value = [];
  }

  async function readPreviewItem(index: number): Promise<StoryPainterLogItem | undefined> {
    const rows = await readPreviewItems([index]);
    return rows[0];
  }

  async function readPreviewItems(indexes: number[]): Promise<StoryPainterLogItem[]> {
    const uniqueIndexes = [...new Set(indexes)].filter(index => Number.isFinite(index));
    if (uniqueIndexes.length === 0) return [];
    const source = dataset.value;
    if (!source) {
      return indexes
        .map(index => previewItems.value.find(item => item.index === index))
        .filter((item): item is StoryPainterLogItem => Boolean(item));
    }
    const missing = uniqueIndexes.filter(index => !rowCache.has(index));
    const chunkStarts = [...new Set(missing.map(index => Math.floor(index / previewReadChunkSize) * previewReadChunkSize))];
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
    const rows = indexes
      .map(index => rowCache.get(index))
      .filter((item): item is StoryPainterLogItem => Boolean(item));
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
    const rows = await source.readAll([...previewColumns]);
    items.value = rows.filter(item => !isDeletedOriginalItem(item)).map(applyCharOverrides);
    fullItemsLoaded.value = true;
  }

  async function* iterPreviewChunks(chunkSize = 2000): AsyncGenerator<StoryPainterLogItem[]> {
    const source = dataset.value;
    if (!source) {
      yield previewItems.value;
      return;
    }
    for await (const chunk of source.iterRows({ columns: [...previewColumns], chunkSize })) {
      const overridden = chunk.filter(item => !isDeletedOriginalItem(item)).map(applyCharOverrides);
      yield buildStoryPainterPreviewItems(overridden, chars.value, exportOptions, (item) =>
        normalizeStoryPainterMessage(item, chars.value, exportOptions, false),
      );
    }
  }

  function originalCharKey(char: StoryPainterChar): string {
    for (const [packed, nextName] of charNameOverrides) {
      const original = unpackStoryPainterNameId(packed);
      if (original.IMUserId === char.IMUserId && nextName === char.name) return packed;
    }
    return packStoryPainterNameId(char);
  }

  function isDeletedOriginalItem(item: StoryPainterLogItem): boolean {
    return deletedOriginalCharKeys.has(packStoryPainterNameId(item));
  }

  function applyCharOverrides(item: StoryPainterLogItem): StoryPainterLogItem {
    if (charNameOverrides.size === 0) return item;
    let next = item;
    for (const [packed, nextName] of charNameOverrides) {
      const { name: oldName, IMUserId } = unpackStoryPainterNameId(packed);
      next = renameStoryPainterChar(next === item ? [item] : [next], { name: oldName, IMUserId, role: '角色', color: '' }, nextName)[0] ?? next;
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

  function readPreviewChunk(source: StoryPainterParquetDataset, start: number, sourceVersion: number): Promise<void> {
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
      const rows = await source.readRows(start, Math.min(start + previewReadChunkSize, source.numRows), [...previewColumns]);
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
      rows.forEach((item) => {
        if (isDeletedOriginalItem(item)) return;
        const overridden = applyCharOverrides(item);
        if (overridden.index !== undefined) {
          setCachedRow(overridden.index, overridden);
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
    visibleIndexes,
    chars,
    swatches,
    mode,
    editorHighlight,
    trgIsAddVoiceMark,
    forumOptions,
    exportOptions,
    pcMap,
    previewCount,
    setItems,
    setDataset,
    setAllItemsFromEditor,
    refreshPreview,
    readPreviewItem,
    readPreviewItems,
    ensureFullItems,
    iterPreviewChunks,
    updateChar,
    renameChar,
    deleteChar,
    refreshSwatches,
    colorByItem,
    hidden,
  };
}

function syncChars(current: StoryPainterChar[], next: StoryPainterChar[]): StoryPainterChar[] {
  const currentMap = new Map(current.map((item) => [packStoryPainterNameId(item), item]));
  return next.map((item, index) => ({
    ...item,
    color: currentMap.get(packStoryPainterNameId(item))?.color || item.color || storyPainterPalette[index % storyPainterPalette.length],
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

function unpackStoryPainterNameId(value: string): { name: string; IMUserId: string } {
  const separatorIndex = value.lastIndexOf('-');
  if (separatorIndex < 0) return { name: value, IMUserId: '' };
  return {
    name: value.slice(0, separatorIndex),
    IMUserId: value.slice(separatorIndex + 1),
  };
}
