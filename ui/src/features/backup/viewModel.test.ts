import type { Config, FileItem } from '@/api';
import {
  buildBackupConfigPayload,
  buildBackupFilenamePreview,
  describeBackupSelection,
  formatBackupSelection,
  formatCleanTriggers,
  getDefaultBatchDeleteNames,
  normalizeBackupConfig,
  parseBackupSelection,
  parseCleanTriggers,
} from './viewModel.js';
import { expect, it } from 'vitest';

it('备份文件列表转换与删除选择', async () => {
  const config: Config = {
    autoBackupEnable: true,
    autoBackupTime: ' @every 12h ',
    autoBackupSelection: 0b100101,
    backupCleanStrategy: 2,
    backupCleanKeepCount: 0,
    backupCleanKeepDur: ' 720h ',
    backupCleanTrigger: 3,
    backupCleanCron: ' 0 0 * * * ',
  };

  expect(parseBackupSelection(0)).toEqual(['base']);
  expect(parseBackupSelection(0b100101)).toEqual(['base', 'js', 'helpdoc', 'image']);
  expect(formatBackupSelection(['image', 'base', 'js', 'image'])).toBe(0b100001);
  expect(describeBackupSelection(0b111111)).toEqual([
    '基础',
    'JS 插件',
    '牌堆',
    '帮助文档',
    '敏感词库',
    '人名信息',
    '图片',
  ]);
  expect(describeBackupSelection(-1)).toEqual([]);

  expect(parseCleanTriggers(3)).toEqual(['cron', 'afterAutoBackup']);
  expect(formatCleanTriggers(['afterAutoBackup'])).toBe(2);

  expect(normalizeBackupConfig(config)).toEqual({
    autoBackupEnable: true,
    autoBackupTime: '@every 12h',
    autoBackupSelection: 0b100101,
    autoBackupSelectionList: ['base', 'js', 'helpdoc', 'image'],
    backupCleanStrategy: 2,
    backupCleanKeepCount: 1,
    backupCleanKeepDur: '720h',
    backupCleanTrigger: 3,
    backupCleanTriggers: ['cron', 'afterAutoBackup'],
    backupCleanCron: '0 0 * * *',
  });

  expect(
    buildBackupConfigPayload({
      ...normalizeBackupConfig(config),
      autoBackupSelectionList: ['base', 'deck', 'censor'],
      backupCleanTriggers: ['afterAutoBackup'],
    })
  ).toEqual({
    autoBackupEnable: true,
    autoBackupTime: '@every 12h',
    autoBackupSelection: 0b001010,
    backupCleanStrategy: 2,
    backupCleanKeepCount: 1,
    backupCleanKeepDur: '720h',
    backupCleanTrigger: 2,
    backupCleanCron: '0 0 * * *',
  });

  expect(buildBackupFilenamePreview('260521_123456', 0b101, false)).toBe(
    'bak_260521_123456_r5_<随机值>.zip'
  );
  expect(buildBackupFilenamePreview('260521_123456', 0b101, true)).toBe(
    'bak_260521_123456_auto_r5_<随机值>.zip'
  );

  const files: FileItem[] = Array.from({ length: 7 }, (_, index) => ({
    name: `bak-${index}`,
    fileSize: index * 100,
    selection: index,
  }));
  expect(getDefaultBatchDeleteNames(files)).toEqual(['bak-5', 'bak-6']);
});
