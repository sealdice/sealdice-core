import {
  buildBanAddPayload,
  buildBanListPayload,
  getBanRankMeta,
  isBanImportFileAccepted,
  normalizeBanConfig,
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

assertDeepEqual(getBanRankMeta(-30), {
  label: '禁止',
  tagType: 'error',
});
assertDeepEqual(getBanRankMeta(-10), {
  label: '警告',
  tagType: 'warning',
});
assertDeepEqual(getBanRankMeta(30), {
  label: '信任',
  tagType: 'success',
});
assertDeepEqual(getBanRankMeta(0), {
  label: '常规',
  tagType: 'default',
});

assertDeepEqual(buildBanListPayload({
  page: 2,
  pageSize: 30,
  keyword: 'QQ:1001',
  ranks: [-30, -10],
  sortBy: 'score',
}), {
  page: 2,
  pageSize: 30,
  keyword: 'QQ:1001',
  filter: {
    orderByBanTime: false,
    orderByScore: true,
    ranks: [-30, -10],
  },
});

assertDeepEqual(buildBanAddPayload({
  id: ' QQ:1001 ',
  rank: -30,
  name: ' 测试用户 ',
  reason: '',
}), {
  id: 'QQ:1001',
  rank: -30,
  name: '测试用户',
  reason: '骰主后台设置',
});

assertEqual(isBanImportFileAccepted('ban.json'), true);
assertEqual(isBanImportFileAccepted('ban.JSON'), true);
assertEqual(isBanImportFileAccepted('ban.txt'), false);

assertDeepEqual(normalizeBanConfig({
  thresholdWarn: 0,
  thresholdBan: 0,
  autoBanMinutes: 0,
  banBehaviorRefuseReply: true,
  banBehaviorRefuseInvite: true,
  banBehaviorQuitLastPlace: false,
  banBehaviorQuitPlaceImmediately: false,
  banBehaviorQuitIfAdmin: false,
  banBehaviorQuitIfAdminSilentIfNotAdmin: false,
  scoreReducePerMinute: 1,
  scoreGroupMuted: 100,
  scoreGroupKicked: 200,
  scoreTooManyCommand: 100,
  jointScorePercentOfGroup: 0.5,
  jointScorePercentOfInviter: 0.3,
}), {
  thresholdWarn: 100,
  thresholdBan: 200,
  autoBanMinutes: 0,
  banBehaviorRefuseReply: true,
  banBehaviorRefuseInvite: true,
  banBehaviorQuitLastPlace: false,
  banBehaviorQuitPlaceImmediately: false,
  banBehaviorQuitIfAdmin: false,
  banBehaviorQuitIfAdminSilentIfNotAdmin: false,
  scoreReducePerMinute: 1,
  scoreGroupMuted: 100,
  scoreGroupKicked: 200,
  scoreTooManyCommand: 100,
  jointScorePercentOfGroup: 0.5,
  jointScorePercentOfInviter: 0.3,
});
