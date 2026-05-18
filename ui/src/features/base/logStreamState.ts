import type { BaseLogItem } from './logStream';

export function applyLogSnapshot(_current: BaseLogItem[], items?: BaseLogItem[] | null): BaseLogItem[] {
  return items ? [...items] : [];
}

export function applyLogAppend(
  current: BaseLogItem[],
  item?: BaseLogItem | null,
  limit = 500,
): BaseLogItem[] {
  if (!item) return current;
  return [...current, item].slice(-limit);
}
