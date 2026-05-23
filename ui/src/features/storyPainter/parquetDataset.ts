import type { AsyncBuffer, FileMetaData } from 'hyparquet';
import type { StoryPainterLogItem } from './types';

export type StoryPainterParquetColumn =
  | 'id'
  | 'nickname'
  | 'IMUserId'
  | 'time'
  | 'message'
  | 'isDice'
  | 'commandId'
  | 'commandInfo'
  | 'uniformId';

export type StoryPainterParquetRow = Partial<Record<StoryPainterParquetColumn, unknown>>;

export interface StoryPainterRowReadOptions {
  columns?: StoryPainterParquetColumn[];
  chunkSize?: number;
}

export interface StoryPainterParquetDataset {
  numRows: number;
  blob?: Blob;
  readRows(start: number, end: number, columns?: StoryPainterParquetColumn[]): Promise<StoryPainterLogItem[]>;
  iterRows(options?: StoryPainterRowReadOptions): AsyncGenerator<StoryPainterLogItem[]>;
  readAll(columns?: StoryPainterParquetColumn[]): Promise<StoryPainterLogItem[]>;
}

const defaultColumns: StoryPainterParquetColumn[] = [
  'id',
  'nickname',
  'IMUserId',
  'time',
  'message',
  'isDice',
  'commandId',
  'commandInfo',
  'uniformId',
];

export async function createStoryPainterParquetDataset(blob: Blob): Promise<StoryPainterParquetDataset> {
  const [{ parquetMetadataAsync, parquetRead }, { compressors }] = await Promise.all([
    import('hyparquet'),
    import('hyparquet-compressors'),
  ]);
  const asyncBuffer = blobAsyncBuffer(blob);
  const metadata = await parquetMetadataAsync(asyncBuffer, {});
  const numRows = toNumber(metadata.num_rows, 0);

  async function readRows(
    start: number,
    end: number,
    columns: StoryPainterParquetColumn[] = defaultColumns,
  ): Promise<StoryPainterLogItem[]> {
    const rowStart = clampRow(start, numRows);
    const rowEnd = clampRow(end, numRows);
    if (rowEnd <= rowStart) return [];
    const rows = await readParquetRows({
      parquetRead,
      file: asyncBuffer,
      metadata,
      compressors,
      rowStart,
      rowEnd,
      columns,
    });
    return bufferToStoryPainterRows(rows, columns, rowStart);
  }

  async function* iterRows(options: StoryPainterRowReadOptions = {}): AsyncGenerator<StoryPainterLogItem[]> {
    const chunkSize = Math.max(1, options.chunkSize ?? 2000);
    for (let start = 0; start < numRows; start += chunkSize) {
      yield await readRows(start, Math.min(start + chunkSize, numRows), options.columns);
    }
  }

  return {
    numRows,
    blob,
    readRows,
    iterRows,
    readAll(columns?: StoryPainterParquetColumn[]) {
      return readRows(0, numRows, columns);
    },
  };
}

export function createMemoryParquetDataset(rows: StoryPainterParquetRow[]): StoryPainterParquetDataset {
  const copy = [...rows];
  const numRows = copy.length;

  async function readRows(
    start: number,
    end: number,
    columns: StoryPainterParquetColumn[] = defaultColumns,
  ): Promise<StoryPainterLogItem[]> {
    const rowStart = clampRow(start, numRows);
    const rowEnd = clampRow(end, numRows);
    return bufferToStoryPainterRows(copy.slice(rowStart, rowEnd), columns, rowStart);
  }

  async function* iterRows(options: StoryPainterRowReadOptions = {}): AsyncGenerator<StoryPainterLogItem[]> {
    const chunkSize = Math.max(1, options.chunkSize ?? 2000);
    for (let start = 0; start < numRows; start += chunkSize) {
      yield await readRows(start, Math.min(start + chunkSize, numRows), options.columns);
    }
  }

  return {
    numRows,
    readRows,
    iterRows,
    readAll(columns?: StoryPainterParquetColumn[]) {
      return readRows(0, numRows, columns);
    },
  };
}

export function bufferToStoryPainterRows(
  rows: StoryPainterParquetRow[],
  columns: StoryPainterParquetColumn[] = defaultColumns,
  startIndex = 0,
): StoryPainterLogItem[] {
  const selected = new Set(columns);
  return rows.map((row, index) => mapParquetRowToStoryPainterItem(row, startIndex + index, selected));
}

export function mapParquetRowToStoryPainterItem(
  row: StoryPainterParquetRow,
  index: number,
  columns = new Set<StoryPainterParquetColumn>(defaultColumns),
): StoryPainterLogItem {
  return {
    id: readColumn(columns, row, 'id') ? toNumber(row.id, index + 1) : index + 1,
    nickname: readColumn(columns, row, 'nickname') ? String(row.nickname ?? '') : '',
    IMUserId: readColumn(columns, row, 'IMUserId') ? String(row.IMUserId ?? '') : '',
    time: readColumn(columns, row, 'time') ? toNumber(row.time, 0) : 0,
    message: readColumn(columns, row, 'message') ? String(row.message ?? '') : '',
    isDice: readColumn(columns, row, 'isDice') ? Boolean(row.isDice) : false,
    commandId: readColumn(columns, row, 'commandId') ? toNumber(row.commandId, 0) : 0,
    commandInfo: readColumn(columns, row, 'commandInfo') ? parseCommandInfo(row.commandInfo) : undefined,
    uniformId: readColumn(columns, row, 'uniformId') && row.uniformId ? String(row.uniformId) : undefined,
    index,
    version: 105,
  };
}

function readColumn(columns: Set<StoryPainterParquetColumn>, row: StoryPainterParquetRow, column: StoryPainterParquetColumn): boolean {
  return columns.has(column) && column in row;
}

async function readParquetRows(options: {
  parquetRead: typeof import('hyparquet')['parquetRead'];
  file: AsyncBuffer;
  metadata: FileMetaData;
  compressors: unknown;
  rowStart: number;
  rowEnd: number;
  columns: StoryPainterParquetColumn[];
}): Promise<StoryPainterParquetRow[]> {
  return await new Promise((resolve, reject) => {
    options.parquetRead({
      file: options.file,
      metadata: options.metadata,
      compressors: options.compressors as never,
      rowStart: options.rowStart,
      rowEnd: options.rowEnd,
      columns: options.columns,
      rowFormat: 'object',
      onComplete: rows => resolve(rows as StoryPainterParquetRow[]),
    }).catch(reject);
  });
}

function blobAsyncBuffer(blob: Blob): AsyncBuffer {
  return {
    byteLength: blob.size,
    slice(start: number, end?: number) {
      return blob.slice(start, end).arrayBuffer();
    },
  };
}

function clampRow(value: number, numRows: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(numRows, Math.trunc(value)));
}

function toNumber(value: unknown, fallback: number): number {
  if (typeof value === 'number') return value;
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'string' && value.trim() !== '') {
    const n = Number(value);
    return Number.isNaN(n) ? fallback : n;
  }
  return fallback;
}

function parseCommandInfo(value: unknown): unknown {
  if (typeof value !== 'string') return value;
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  try {
    return JSON.parse(trimmed);
  } catch {
    return trimmed;
  }
}
