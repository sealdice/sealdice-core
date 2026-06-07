import type { AddReqBody, BanConfig } from '@/api';

export type BanListSortKey = 'time' | 'score';

export type BanListQueryModel = {
  page: number;
  pageSize: number;
  keyword: string;
  ranks: number[];
  sortBy: BanListSortKey;
};

export type BanAddFormModel = {
  id: string;
  rank: number;
  name: string;
  reason: string;
};

export function createDefaultBanListQuery(): BanListQueryModel {
  return {
    page: 1,
    pageSize: 10,
    keyword: '',
    ranks: [-30, -10, 30, 0],
    sortBy: 'time',
  };
}

export function createDefaultBanAddForm(): BanAddFormModel {
  return {
    id: '',
    rank: -30,
    name: '',
    reason: '',
  };
}

export function getBanRankMeta(rank: number) {
  if (rank === -30) {
    return { label: '禁止', tagType: 'error' as const };
  }
  if (rank === -10) {
    return { label: '警告', tagType: 'warning' as const };
  }
  if (rank === 30) {
    return { label: '信任', tagType: 'success' as const };
  }
  return { label: '常规', tagType: 'default' as const };
}

export function buildBanListPayload(query: BanListQueryModel) {
  return {
    page: query.page,
    pageSize: query.pageSize,
    keyword: query.keyword.trim(),
    filter: {
      orderByBanTime: query.sortBy === 'time',
      orderByScore: query.sortBy === 'score',
      ranks: query.ranks,
    },
  };
}

export function buildBanAddPayload(form: BanAddFormModel): AddReqBody {
  return {
    id: form.id.trim(),
    rank: form.rank,
    name: form.name.trim(),
    reason: form.reason.trim() || '骰主后台设置',
  };
}

export function isBanImportFileAccepted(filename: string) {
  return filename.toLowerCase().endsWith('.json');
}

export function normalizeBanConfig(config: BanConfig): BanConfig {
  return {
    ...config,
    thresholdWarn: config.thresholdWarn || 100,
    thresholdBan: config.thresholdBan || 200,
  };
}
