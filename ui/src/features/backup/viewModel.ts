import type { Config, ConfigWritable, FileItem } from '@/api';

export type BackupSelectionKey = 'base' | 'js' | 'deck' | 'helpdoc' | 'censor' | 'name' | 'image';
export type BackupCleanTriggerKey = 'cron' | 'afterAutoBackup';

export type BackupConfigDraft = ConfigWritable & {
  autoBackupSelectionList: BackupSelectionKey[];
  backupCleanTriggers: BackupCleanTriggerKey[];
};

export type BackupSelectionOption = {
  key: BackupSelectionKey;
  label: string;
  bit: number;
  disabled?: boolean;
};

export const BACKUP_SELECTION_OPTIONS: BackupSelectionOption[] = [
  { key: 'base', label: '基础（含自定义回复）', bit: 0, disabled: true },
  { key: 'js', label: 'JS 插件', bit: 1 << 0 },
  { key: 'deck', label: '牌堆', bit: 1 << 1 },
  { key: 'helpdoc', label: '帮助文档', bit: 1 << 2 },
  { key: 'censor', label: '敏感词库', bit: 1 << 3 },
  { key: 'name', label: '人名信息', bit: 1 << 4 },
  { key: 'image', label: '图片', bit: 1 << 5 },
];

const CLEAN_TRIGGER_BITS: Record<BackupCleanTriggerKey, number> = {
  cron: 1 << 0,
  afterAutoBackup: 1 << 1,
};

function cleanText(value: unknown, fallback = '') {
  return typeof value === 'string' ? value.trim() : fallback;
}

function cleanPositiveInt(value: unknown, fallback: number) {
  const numberValue = Number(value);
  if (!Number.isFinite(numberValue) || numberValue < 1) return fallback;
  return Math.trunc(numberValue);
}

export function parseBackupSelection(selection: number): BackupSelectionKey[] {
  const selected: BackupSelectionKey[] = ['base'];
  for (const option of BACKUP_SELECTION_OPTIONS) {
    if (option.key === 'base') continue;
    if ((selection & option.bit) !== 0) {
      selected.push(option.key);
    }
  }
  return selected;
}

export function formatBackupSelection(selections: BackupSelectionKey[]): number {
  const selected = new Set(selections);
  return BACKUP_SELECTION_OPTIONS.reduce((mask, option) => {
    if (option.key === 'base') return mask;
    return selected.has(option.key) ? mask | option.bit : mask;
  }, 0);
}

export function describeBackupSelection(selection: number): string[] {
  if (selection < 0) return [];
  const selected = new Set(parseBackupSelection(selection));
  return BACKUP_SELECTION_OPTIONS
    .filter(option => selected.has(option.key))
    .map(option => option.label.replace('（含自定义回复）', ''));
}

export function parseCleanTriggers(trigger: number): BackupCleanTriggerKey[] {
  return (Object.entries(CLEAN_TRIGGER_BITS) as Array<[BackupCleanTriggerKey, number]>)
    .filter(([, bit]) => (trigger & bit) !== 0)
    .map(([key]) => key);
}

export function formatCleanTriggers(triggers: BackupCleanTriggerKey[]): number {
  const selected = new Set(triggers);
  return (Object.entries(CLEAN_TRIGGER_BITS) as Array<[BackupCleanTriggerKey, number]>).reduce((mask, [key, bit]) => {
    return selected.has(key) ? mask | bit : mask;
  }, 0);
}

export function normalizeBackupConfig(config: Config): BackupConfigDraft {
  const autoBackupSelection = Number(config.autoBackupSelection ?? 0);
  const backupCleanTrigger = Number(config.backupCleanTrigger ?? 0);
  const backupCleanStrategy = [0, 1, 2].includes(config.backupCleanStrategy) ? config.backupCleanStrategy : 0;

  return {
    autoBackupEnable: Boolean(config.autoBackupEnable),
    autoBackupTime: cleanText(config.autoBackupTime, '@every 12h') || '@every 12h',
    autoBackupSelection,
    autoBackupSelectionList: parseBackupSelection(autoBackupSelection),
    backupCleanStrategy,
    backupCleanKeepCount: cleanPositiveInt(config.backupCleanKeepCount, 1),
    backupCleanKeepDur: cleanText(config.backupCleanKeepDur),
    backupCleanTrigger,
    backupCleanTriggers: parseCleanTriggers(backupCleanTrigger),
    backupCleanCron: cleanText(config.backupCleanCron),
  };
}

export function buildBackupConfigPayload(draft: BackupConfigDraft): ConfigWritable {
  return {
    autoBackupEnable: draft.autoBackupEnable,
    autoBackupTime: cleanText(draft.autoBackupTime, '@every 12h') || '@every 12h',
    autoBackupSelection: formatBackupSelection(draft.autoBackupSelectionList),
    backupCleanStrategy: draft.backupCleanStrategy,
    backupCleanKeepCount: cleanPositiveInt(draft.backupCleanKeepCount, 1),
    backupCleanKeepDur: cleanText(draft.backupCleanKeepDur),
    backupCleanTrigger: formatCleanTriggers(draft.backupCleanTriggers),
    backupCleanCron: cleanText(draft.backupCleanCron),
  };
}

export function buildBackupFilenamePreview(timestamp: string, selection: number, auto: boolean) {
  return `bak_${timestamp}_${auto ? 'auto_' : ''}r${selection.toString(16)}_<随机值>.zip`;
}

export function getDefaultBatchDeleteNames(items: FileItem[], keepLatest = 5): string[] {
  return items.slice(keepLatest).map(item => item.name);
}
