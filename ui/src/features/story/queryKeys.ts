import {
  getSdApiV2StoryBackupListQueryKey,
  getSdApiV2StoryInfoQueryKey,
} from '@/api';

export const storyInfoQueryKey = () => getSdApiV2StoryInfoQueryKey();

export const storyBackupListQueryKey = () => getSdApiV2StoryBackupListQueryKey();
