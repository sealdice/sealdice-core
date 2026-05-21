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

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  const normalize = (value: unknown): string => {
    if (Array.isArray(value)) {
      return `[${value.map(normalize).join(',')}]`;
    }
    if (value && typeof value === 'object') {
      const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b));
      return `{${entries.map(([key, entry]) => `${key}:${normalize(entry)}`).join(',')}}`;
    }
    return JSON.stringify(value);
  };
  if (normalize(actual) !== normalize(expected)) {
    throw new Error(`expected ${normalize(expected)}, got ${normalize(actual)}`);
  }
};

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

assertDeepEqual(parseBackupSelection(0), ['base']);
assertDeepEqual(parseBackupSelection(0b100101), ['base', 'js', 'helpdoc', 'image']);
assertEqual(formatBackupSelection(['image', 'base', 'js', 'image']), 0b100001);
assertDeepEqual(describeBackupSelection(0b111111), ['基础', 'JS 插件', '牌堆', '帮助文档', '敏感词库', '人名信息', '图片']);
assertDeepEqual(describeBackupSelection(-1), []);

assertDeepEqual(parseCleanTriggers(3), ['cron', 'afterAutoBackup']);
assertEqual(formatCleanTriggers(['afterAutoBackup']), 2);

assertDeepEqual(normalizeBackupConfig(config), {
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

assertDeepEqual(buildBackupConfigPayload({
  ...normalizeBackupConfig(config),
  autoBackupSelectionList: ['base', 'deck', 'censor'],
  backupCleanTriggers: ['afterAutoBackup'],
}), {
  autoBackupEnable: true,
  autoBackupTime: '@every 12h',
  autoBackupSelection: 0b001010,
  backupCleanStrategy: 2,
  backupCleanKeepCount: 1,
  backupCleanKeepDur: '720h',
  backupCleanTrigger: 2,
  backupCleanCron: '0 0 * * *',
});

assertEqual(buildBackupFilenamePreview('260521_123456', 0b101, false), 'bak_260521_123456_r5_<随机值>.zip');
assertEqual(buildBackupFilenamePreview('260521_123456', 0b101, true), 'bak_260521_123456_auto_r5_<随机值>.zip');

const files: FileItem[] = Array.from({ length: 7 }, (_, index) => ({
  name: `bak-${index}`,
  fileSize: index * 100,
  selection: index,
}));
assertDeepEqual(getDefaultBatchDeleteNames(files), ['bak-5', 'bak-6']);
