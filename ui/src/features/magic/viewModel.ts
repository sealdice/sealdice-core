export interface MagicInspectSummaryInput {
  kind: 'sqlite' | 'mysql' | 'postgres';
  stage: 'current' | 'legacy_v146' | 'unknown_legacy';
  canDirectMigrate: boolean;
  requiresV150Upgrade: boolean;
  requiresSqliteRepair: boolean;
  messages: string[];
  fingerprint: {
    tables: string[];
  };
}

export interface MagicInspectSummary {
  headline: string;
  tone: 'success' | 'warning' | 'error';
  stageText: string;
  nextActionText: string;
  tableCount: number;
  tablePreview: string;
}

export function normalizeMagicInspectResult(input: {
  kind: string;
  stage: string;
  canDirectMigrate: boolean;
  requiresV150Upgrade: boolean;
  requiresSqliteRepair: boolean;
  messages?: string[] | null;
  fingerprint?: {
    tables?: string[] | null;
  } | null;
}): MagicInspectSummaryInput {
  return {
    kind: normalizeKind(input.kind),
    stage: normalizeStage(input.stage),
    canDirectMigrate: input.canDirectMigrate,
    requiresV150Upgrade: input.requiresV150Upgrade,
    requiresSqliteRepair: input.requiresSqliteRepair,
    messages: input.messages ?? [],
    fingerprint: {
      tables: input.fingerprint?.tables ?? [],
    },
  };
}

export function getMagicInspectSummary(input: MagicInspectSummaryInput): MagicInspectSummary {
  const tableCount = input.fingerprint.tables.length;
  const tablePreview = input.fingerprint.tables.slice(0, 6).join('、');

  if (input.stage === 'current' && input.canDirectMigrate) {
    return {
      headline: '源结构已就绪',
      tone: 'success',
      stageText: '当前标准结构',
      nextActionText: '可以直接进入目标库选择和迁移任务生成。',
      tableCount,
      tablePreview,
    };
  }

  if (input.stage === 'legacy_v146' && input.requiresV150Upgrade) {
    return {
      headline: '检测到 1.4.6 旧结构',
      tone: 'warning',
      stageText: '1.4.6 attrs 旧结构',
      nextActionText: '必须先在 SQLite 工作副本上升级到 1.5.0，再继续迁移。',
      tableCount,
      tablePreview,
    };
  }

  return {
    headline: '源结构暂不支持直接迁移',
    tone: 'error',
    stageText: '未知旧结构',
    nextActionText: '请先使用官方程序完成原地升级，再重新执行迁移向导。',
    tableCount,
    tablePreview,
  };
}

function normalizeKind(kind: string): MagicInspectSummaryInput['kind'] {
  if (kind === 'mysql' || kind === 'postgres' || kind === 'sqlite') return kind;
  return 'sqlite';
}

function normalizeStage(stage: string): MagicInspectSummaryInput['stage'] {
  if (stage === 'current' || stage === 'legacy_v146' || stage === 'unknown_legacy') return stage;
  return 'unknown_legacy';
}
