import type { BaseLogEntry } from './logStream';

export function applyLogDisplayUpdate(
  current: BaseLogEntry[],
  source: BaseLogEntry[],
  autoRefresh: boolean
): BaseLogEntry[] {
  return autoRefresh ? [...source] : current;
}
