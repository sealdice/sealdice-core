import {
  getSdApiV2StoryBackupListQueryKey,
  getSdApiV2StoryInfoQueryKey,
} from '@/api';
import { storyBackupListQueryKey, storyInfoQueryKey } from './queryKeys';

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(storyInfoQueryKey(), getSdApiV2StoryInfoQueryKey());
assertDeepEqual(storyBackupListQueryKey(), getSdApiV2StoryBackupListQueryKey());
