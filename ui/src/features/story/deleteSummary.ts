type StoryLogSummary = {
  name: string;
  groupId: string;
};

export function summarizeStoryLogs(logs: StoryLogSummary[]): string {
  return `共 ${logs.length} 份日志：${logs
    .map(log => `${log.name}（${log.groupId}）`)
    .join('、')}`;
}
