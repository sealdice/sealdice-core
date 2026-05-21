import type { ReplyFileDetail } from '@/api';

export type ReplyCondition = {
  condType: string;
  matchType: string;
  matchOp?: string;
  value: string | number;
};

export type ReplyMessage = [string, number];

export type ReplyResult = {
  resultType: string;
  delay: number;
  message: ReplyMessage[];
};

export type ReplyTask = {
  enable: boolean;
  conditions: ReplyCondition[];
  results: ReplyResult[];
};

export type ReplyFileDraft = {
  enable: boolean;
  interval: number;
  name: string;
  author: string[];
  version: string;
  createTimestamp: number;
  updateTimestamp: number;
  desc: string;
  storeID: string;
  conditions: ReplyCondition[];
  items: ReplyTask[];
  filename: string;
  itemCount: number;
};

type ReplyApiCondition = {
  condType?: string;
  matchType?: string;
  matchOp?: string;
  value?: unknown;
};

type ReplyApiResult = {
  resultType?: string;
  delay?: unknown;
  message?: unknown[] | null;
};

type ReplyApiTask = {
  enable?: boolean;
  conditions?: unknown[] | null;
  results?: unknown[] | null;
};

export function cloneReplyTask(item: ReplyTask): ReplyTask {
  return {
    enable: item.enable,
    conditions: item.conditions.map(condition => ({ ...condition })),
    results: item.results.map(result => ({
      resultType: result.resultType,
      delay: result.delay,
      message: result.message.map(messageItem => [messageItem[0], messageItem[1]]),
    })),
  };
}

export function cloneReplyFileDraft(item: ReplyFileDraft): ReplyFileDraft {
  return {
    ...item,
    author: [...item.author],
    conditions: item.conditions.map(condition => ({ ...condition })),
    items: item.items.map(task => cloneReplyTask(task)),
  };
}

export function normalizeReplyFileDetail(detail: ReplyFileDetail): ReplyFileDraft {
  return {
    enable: detail.enable,
    interval: Number(detail.interval ?? 0),
    name: String(detail.name ?? ''),
    author: (detail.author ?? []).map(item => String(item)),
    version: String(detail.version ?? ''),
    createTimestamp: Number(detail.createTimestamp ?? 0),
    updateTimestamp: Number(detail.updateTimestamp ?? 0),
    desc: String(detail.desc ?? ''),
    storeID: String(detail.storeID ?? ''),
    conditions: normalizeConditions(detail.conditions as unknown[] | null | undefined),
    items: [],
    filename: String(detail.filename ?? ''),
    itemCount: Number(detail.itemCount ?? 0),
  };
}

export function normalizeReplyTask(item: unknown): ReplyTask {
  const raw = (item ?? {}) as ReplyApiTask;
  return {
    enable: raw.enable !== false,
    conditions: normalizeConditions(raw.conditions),
    results: normalizeResults(raw.results),
  };
}

export function normalizeConditions(items: unknown[] | null | undefined): ReplyCondition[] {
  return (items ?? []).map(item => normalizeCondition(item));
}

export function normalizeCondition(item: unknown): ReplyCondition {
  const raw = (item ?? {}) as ReplyApiCondition;
  return {
    condType: String(raw.condType ?? 'textMatch'),
    matchType: String(raw.matchType ?? 'matchExact'),
    ...(raw.matchOp ? { matchOp: String(raw.matchOp) } : {}),
    value: typeof raw.value === 'number' ? raw.value : String(raw.value ?? ''),
  };
}

export function normalizeResults(items: unknown[] | null | undefined): ReplyResult[] {
  return (items ?? []).map(item => {
    const raw = (item ?? {}) as ReplyApiResult;
    return {
      resultType: String(raw.resultType ?? 'replyToSender'),
      delay: Number(raw.delay ?? 0),
      message: normalizeMessages(raw.message),
    };
  });
}

export function normalizeMessages(items: unknown[] | null | undefined): ReplyMessage[] {
  return (items ?? []).map(item => {
    const parts = Array.isArray(item) ? item : [];
    return [String(parts[0] ?? ''), Number(parts[1] ?? 1)];
  });
}

export function toApiReplyConfig(draft: ReplyFileDraft) {
  return {
    enable: draft.enable,
    interval: Number(draft.interval) || 0,
    items: draft.items.filter(Boolean).map(item => ({
      enable: item.enable,
      conditions: item.conditions.map(condition => {
        if (condition.condType === 'textLenLimit') {
          return {
            ...condition,
            value: Number(condition.value) || 0,
          };
        }
        return condition;
      }),
      results: item.results.map(result => ({
        ...result,
        delay: Number(result.delay) || 0,
      })),
    })),
    name: draft.name,
    author: draft.author,
    version: draft.version,
    createTimestamp: draft.createTimestamp,
    updateTimestamp: draft.updateTimestamp,
    desc: draft.desc,
    storeID: draft.storeID,
    filename: draft.filename,
    conditions: draft.conditions.map(condition => {
      if (condition.condType === 'textLenLimit') {
        return {
          ...condition,
          value: Number(condition.value) || 0,
        };
      }
      return condition;
    }),
  };
}
