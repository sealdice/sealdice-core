import { getMagicInspectSummary } from './viewModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const currentSummary = getMagicInspectSummary({
  kind: 'sqlite',
  stage: 'current',
  canDirectMigrate: true,
  requiresV150Upgrade: false,
  requiresSqliteRepair: false,
  messages: [],
  fingerprint: {
    tables: ['attrs', 'ban_info', 'censor_log', 'endpoint_info', 'group_info', 'group_player_info', 'log_items', 'logs'],
  },
});

assertEqual(currentSummary.headline, '源结构已就绪');
assertEqual(currentSummary.tone, 'success');
assertEqual(currentSummary.stageText, '当前标准结构');
assertEqual(currentSummary.tableCount, 8);

const legacySummary = getMagicInspectSummary({
  kind: 'sqlite',
  stage: 'legacy_v146',
  canDirectMigrate: false,
  requiresV150Upgrade: true,
  requiresSqliteRepair: false,
  messages: ['需要先升级到 1.5.0'],
  fingerprint: {
    tables: ['attrs_group', 'attrs_group_user', 'attrs_user', 'group_info'],
  },
});

assertEqual(legacySummary.headline, '检测到 1.4.6 旧结构');
assertEqual(legacySummary.tone, 'warning');
assertEqual(legacySummary.stageText, '1.4.6 attrs 旧结构');

const unknownSummary = getMagicInspectSummary({
  kind: 'sqlite',
  stage: 'unknown_legacy',
  canDirectMigrate: false,
  requiresV150Upgrade: false,
  requiresSqliteRepair: false,
  messages: [],
  fingerprint: {
    tables: ['attrs_user'],
  },
});

assertEqual(unknownSummary.headline, '源结构暂不支持直接迁移');
assertEqual(unknownSummary.tone, 'error');
assertEqual(unknownSummary.stageText, '未知旧结构');
