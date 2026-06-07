import type { BaseLogItem } from './logStream';

export function applyLogDisplayUpdate(
  current: BaseLogItem[],
  source: BaseLogItem[],
  autoRefresh: boolean,
): BaseLogItem[] {
  return autoRefresh ? [...source] : current;
}
