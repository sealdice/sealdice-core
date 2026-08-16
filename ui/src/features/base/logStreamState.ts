import type { BaseLogEntry, BaseLogItem } from './logStream';

// 快照与追加共用一个自增序列，保证同一次会话内 id 不会重复，也不会随倒序显示变化。
let nextLogId = 0;

export function resetLogIdSequence(): void {
  nextLogId = 0;
}

function toEntry(item: BaseLogItem): BaseLogEntry {
  return { ...item, id: nextLogId++ };
}

export function applyLogSnapshot(
  _current: BaseLogEntry[],
  items?: BaseLogItem[] | null
): BaseLogEntry[] {
  return items ? items.map(toEntry) : [];
}

export function applyLogAppend(
  current: BaseLogEntry[],
  item?: BaseLogItem | null,
  limit = 500
): BaseLogEntry[] {
  if (!item) return current;
  return [...current, toEntry(item)].slice(-limit);
}
