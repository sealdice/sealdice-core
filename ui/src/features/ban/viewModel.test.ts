import {
  buildBanAddPayload,
  buildBanListPayload,
  getBanRankMeta,
  isBanImportFileAccepted,
  normalizeBanConfig,
} from './viewModel.js';
import { expect, it } from 'vitest';

it('黑名单配置和列表数据转换', async () => {
  expect(getBanRankMeta(-30)).toEqual({
    label: '禁止',
    tagType: 'error',
  });
  expect(getBanRankMeta(-10)).toEqual({
    label: '警告',
    tagType: 'warning',
  });
  expect(getBanRankMeta(30)).toEqual({
    label: '信任',
    tagType: 'success',
  });
  expect(getBanRankMeta(0)).toEqual({
    label: '常规',
    tagType: 'default',
  });

  expect(
    buildBanListPayload({
      page: 2,
      pageSize: 30,
      keyword: 'QQ:1001',
      ranks: [-30, -10],
      sortBy: 'score',
    })
  ).toEqual({
    page: 2,
    pageSize: 30,
    keyword: 'QQ:1001',
    filter: {
      orderByBanTime: false,
      orderByScore: true,
      ranks: [-30, -10],
    },
  });

  expect(
    buildBanAddPayload({
      id: ' QQ:1001 ',
      rank: -30,
      name: ' 测试用户 ',
      reason: '',
    })
  ).toEqual({
    id: 'QQ:1001',
    rank: -30,
    name: '测试用户',
    reason: '骰主后台设置',
  });

  expect(isBanImportFileAccepted('ban.json')).toBe(true);
  expect(isBanImportFileAccepted('ban.JSON')).toBe(true);
  expect(isBanImportFileAccepted('ban.txt')).toBe(false);

  expect(
    normalizeBanConfig({
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
      banNotifyIntervalMinutes: -1,
      jointScorePercentOfGroup: 0.5,
      jointScorePercentOfInviter: 0.3,
    })
  ).toEqual({
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
    banNotifyIntervalMinutes: -1,
    jointScorePercentOfGroup: 0.5,
    jointScorePercentOfInviter: 0.3,
  });
});
