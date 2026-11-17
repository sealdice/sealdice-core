import { describe, expect, it } from 'vitest';
import { getSdApiV2StoryBackupListQueryKey, getSdApiV2StoryInfoQueryKey } from '@/api';
import {
  getStoryPageSizeChange,
  setStoryLogsSelected,
  storyBackupListQueryKey,
  storyInfoQueryKey,
  summarizeStoryLogs,
} from './model';

describe('跑团日志模型', () => {
  it('使用一致的 API 查询键', () => {
    expect(storyInfoQueryKey()).toEqual(getSdApiV2StoryInfoQueryKey());
    expect(storyBackupListQueryKey()).toEqual(getSdApiV2StoryBackupListQueryKey());
  });

  it('切换分页大小时返回第一页', () => {
    expect(getStoryPageSizeChange(200)).toEqual({ pageNum: 1, pageSize: 200 });
  });

  it('批量选择与取消选择日志', () => {
    const logs = [{ pitch: true }, { pitch: false }, { pitch: false }];
    setStoryLogsSelected(logs, true);
    expect(logs.every(log => log.pitch)).toBe(true);
    setStoryLogsSelected(logs, false);
    expect(logs.every(log => !log.pitch)).toBe(true);
  });

  it('汇总所选日志名称与所属群组', () => {
    expect(
      summarizeStoryLogs([
        { name: '团本 A', groupId: 'group-a' },
        { name: '团本 B', groupId: 'group-b' },
      ])
    ).toBe('共 2 份日志：团本 A（group-a）、团本 B（group-b）');
  });
});
