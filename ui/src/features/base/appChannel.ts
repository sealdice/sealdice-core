export function formatAppChannel(channel: string | undefined): string {
  if (channel === 'stable') return '正式版';
  if (channel === 'dev') return '开发版';
  return '未知';
}
