import { getSdApiV2StoryBackupListQueryKey, getSdApiV2StoryInfoQueryKey } from '@/api';

export const storyInfoQueryKey = () => getSdApiV2StoryInfoQueryKey();
export const storyBackupListQueryKey = () => getSdApiV2StoryBackupListQueryKey();

export type StorySelectableLog = { pitch?: boolean };

export function setStoryLogsSelected<T extends StorySelectableLog>(
  logs: T[],
  selected: boolean
): void {
  for (const log of logs) log.pitch = selected;
}

export function getStoryPageSizeChange(pageSize: number): { pageNum: number; pageSize: number } {
  return { pageNum: 1, pageSize };
}

export function summarizeStoryLogs(logs: { name: string; groupId: string }[]): string {
  return (
    '共 ' +
    logs.length +
    ' 份日志：' +
    logs.map(log => log.name + '（' + log.groupId + '）').join('、')
  );
}
