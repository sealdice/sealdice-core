import { getSdApiV2StoryLogExportParquet } from '@/api';

export async function fetchStoryLogParquet(logId: number): Promise<Blob> {
  const { data } = await getSdApiV2StoryLogExportParquet({
    query: { logId },
    parseAs: 'blob',
    throwOnError: true,
  });
  if (!(data instanceof Blob)) {
    throw new Error('日志导出响应不是文件');
  }
  return data;
}
