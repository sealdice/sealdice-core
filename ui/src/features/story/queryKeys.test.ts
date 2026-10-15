import { getSdApiV2StoryBackupListQueryKey, getSdApiV2StoryInfoQueryKey } from '@/api';
import { storyBackupListQueryKey, storyInfoQueryKey } from './queryKeys';
import { expect, it } from 'vitest';

it('跑团日志查询键按日志和分页隔离', async () => {
  expect(storyInfoQueryKey()).toEqual(getSdApiV2StoryInfoQueryKey());
  expect(storyBackupListQueryKey()).toEqual(getSdApiV2StoryBackupListQueryKey());
});
