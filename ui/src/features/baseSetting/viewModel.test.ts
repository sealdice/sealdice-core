import {
  buildBaseSettingPatch,
  buildBaseSettingSearchIndex,
  buildExtDefaultSettingsView,
  buildBaseSettingStringListOptions,
  filterExtDefaultSettingsView,
  isCronExpressionValid,
  isEveryDuration,
  isRateFormatValid,
  validateBaseSettingPatchFormats,
  getExtDefaultSettingPage,
  getBaseSettingFieldFeedback,
  getBaseSettingFieldLayout,
  getExtDefaultSettingModifiedCount,
  isBaseSettingGroupWide,
  normalizeBaseSettingSchema,
  normalizeBaseSettingValue,
  normalizeMasterListValues,
  normalizeStringListValues,
  searchBaseSettingFields,
  searchExtDefaultSettingsView,
  sortExtDefaultSettingsView,
} from './viewModel.js';
import type { BaseSettingExtDefaultSettingItem, BaseSettingValueResp } from '@/api';
import type { BaseSettingFieldSchema, BaseSettingSchema } from './viewModel';
import { expect, it } from 'vitest';

it('规范化设置描述并按字段与分类搜索', () => {
  const schema = normalizeBaseSettingSchema({
    tabs: [
      {
        id: 'platform-special',
        title: '平台特殊配置',
        groups: [
          {
            id: 'qq-channel',
            title: 'QQ平台配置',
            fields: [
              {
                id: 'qq-enable-poke',
                key: 'QQEnablePoke',
                label: '启用戳一戳',
                kind: 'boolean',
                keywords: ['QQ特性', '戳一戳'],
              },
            ],
          },
        ],
      },
    ],
  } satisfies BaseSettingSchema);

  const index = buildBaseSettingSearchIndex(schema);
  expect(index.length).toBe(1);
  expect(searchBaseSettingFields(index, '戳一戳')[0]?.fieldId).toBe('qq-enable-poke');
  expect(searchBaseSettingFields(index, '平台')[0]?.tabId).toBe('platform-special');
});

it('设置字段按类型派生布局与提示', () => {
  expect(isBaseSettingGroupWide('ext-default-settings')).toBe(true);
  expect(isBaseSettingGroupWide('upgrade')).toBe(true);
  expect(isBaseSettingGroupWide('mail')).toBe(false);
  expect(
    getBaseSettingFieldLayout({ kind: 'boolean' } satisfies Pick<BaseSettingFieldSchema, 'kind'>)
  ).toBe('inline');
  expect(
    getBaseSettingFieldLayout({ kind: 'text' } satisfies Pick<BaseSettingFieldSchema, 'kind'>)
  ).toBe('auto');
  expect(
    getBaseSettingFieldLayout({ kind: 'string-list' } satisfies Pick<
      BaseSettingFieldSchema,
      'kind'
    >)
  ).toBe('stacked');
  expect(
    getBaseSettingFieldLayout({ kind: 'upload' } satisfies Pick<BaseSettingFieldSchema, 'kind'>)
  ).toBe('stacked');
  expect(
    getBaseSettingFieldLayout({ kind: 'master-list' } satisfies Pick<
      BaseSettingFieldSchema,
      'kind'
    >)
  ).toBe('stacked');
  expect(
    getBaseSettingFieldFeedback({ key: 'QQEnablePoke' } satisfies Pick<
      BaseSettingFieldSchema,
      'key' | 'hint'
    >)
  ).toBe(
    '启用前请确认你使用的 QQ 连接方式支持该功能，若不支持请关闭该功能来避免日志中出现相关报错。'
  );
  expect(
    getBaseSettingFieldFeedback({ hint: '用于发送通知' } satisfies Pick<
      BaseSettingFieldSchema,
      'key' | 'hint'
    >)
  ).toBe('用于发送通知');
});

it('字符串与骰主列表清理空值并去重', () => {
  expect(buildBaseSettingStringListOptions(['.', '!', '.', '', ' QQ:1 '])).toEqual([
    { label: '.', value: '.' },
    { label: '!', value: '!' },
    { label: 'QQ:1', value: 'QQ:1' },
  ]);
  expect(normalizeMasterListValues(['QQ:1', ' QQ:1 ', '', 'QQ:2', 'QQ:1'])).toEqual([
    'QQ:1',
    'QQ:2',
  ]);
  expect(normalizeStringListValues(['.', ' . ', '', '!', '.'])).toEqual(['.', '!']);
});

it('频率与定时格式拒绝非法值并标出补丁错误字段', () => {
  expect(isEveryDuration('@every 3h')).toBe(true);
  expect(isEveryDuration('@every 1h30m')).toBe(true);
  expect(isEveryDuration('@every abc')).toBe(false);
  expect(isEveryDuration('@EVERY 3h')).toBe(false);
  expect(isEveryDuration('every 3h')).toBe(false);
  expect(isEveryDuration('3h')).toBe(false);
  expect(isRateFormatValid('3')).toBe(true);
  expect(isRateFormatValid('0')).toBe(false);
  expect(isRateFormatValid('-1')).toBe(false);
  expect(isRateFormatValid('@every 1s')).toBe(true);
  expect(isRateFormatValid('abc')).toBe(false);
  expect(isRateFormatValid('')).toBe(false);
  expect(isCronExpressionValid('@every 3h')).toBe(true);
  expect(isCronExpressionValid('@daily')).toBe(true);
  expect(isCronExpressionValid('@weekly')).toBe(true);
  expect(isCronExpressionValid('0 12 * * *')).toBe(true);
  expect(isCronExpressionValid('*/5 * * * *')).toBe(true);
  expect(isCronExpressionValid('1,2,3 0 * * *')).toBe(true);
  expect(isCronExpressionValid('0-30/5 9 * JAN-MAR MON-FRI')).toBe(true);
  expect(isCronExpressionValid('0 0 * * ?')).toBe(true);
  expect(isCronExpressionValid('TZ=Asia/Shanghai 0 12 * * *')).toBe(true);
  expect(isCronExpressionValid('TZ=Not/AZone 0 12 * * *')).toBe(false);
  expect(isCronExpressionValid('0 12 * * 7')).toBe(false);
  expect(isCronExpressionValid('0 12 * *')).toBe(false);
  expect(isCronExpressionValid('0 60 * * *')).toBe(false);
  expect(isCronExpressionValid('0 12 * * 8')).toBe(false);
  expect(isCronExpressionValid('* * * * * *')).toBe(false);
  expect(isCronExpressionValid('0 0 32 * *')).toBe(false);
  expect(isCronExpressionValid('0 0 * 13 *')).toBe(false);
  expect(isCronExpressionValid('JAN 12 * * *')).toBe(false);
  expect(isCronExpressionValid('')).toBe(false);
  expect(validateBaseSettingPatchFormats({})).toEqual([]);
  expect(validateBaseSettingPatchFormats({ mailEnable: true })).toEqual([]);
  expect(
    validateBaseSettingPatchFormats({ personalReplenishRate: 'abc' }).map(error => error.key)
  ).toEqual(['personalReplenishRate']);
  expect(
    validateBaseSettingPatchFormats({ groupReplenishRate: '0', aliveNoticeValue: 'bad cron' }).map(
      error => error.key
    )
  ).toEqual(['groupReplenishRate', 'aliveNoticeValue']);
  expect(
    validateBaseSettingPatchFormats({ personalReplenishRate: '3', aliveNoticeValue: '@every 3h' })
      .length
  ).toBe(0);
});

it('扩展默认设置按变更筛选排序和分页', () => {
  const extInitial: BaseSettingExtDefaultSettingItem[] = [
    {
      name: 'coc7',
      autoActive: true,
      disabledCommand: { ra: false, rc: true },
      loaded: true,
    },
    {
      name: 'fun',
      autoActive: false,
      disabledCommand: { joke: false },
      loaded: true,
    },
    {
      name: 'story',
      autoActive: true,
      disabledCommand: { log: false, note: false, tips: false },
      loaded: true,
    },
  ];

  const extCurrent: BaseSettingExtDefaultSettingItem[] = [
    extInitial[0],
    {
      name: 'fun',
      autoActive: true,
      disabledCommand: { joke: false },
      loaded: true,
    },
    {
      name: 'story',
      autoActive: true,
      disabledCommand: { log: false, note: true, tips: false },
      loaded: true,
    },
  ];

  const extView = buildExtDefaultSettingsView(extCurrent, extInitial);
  expect(extView.length).toBe(3);
  expect(extView[0]?.dirty).toBe(false);
  expect(extView[1]?.dirty).toBe(true);
  expect(extView[1]?.autoActiveDirty).toBe(true);
  expect(extView[2]?.disabledCommandDirty).toBe(true);
  expect(extView[2]?.changedCommands).toEqual(['note']);
  expect(getExtDefaultSettingModifiedCount(extView)).toBe(2);
  expect(filterExtDefaultSettingsView(extView, 'modified').map(item => item.item.name)).toEqual([
    'fun',
    'story',
  ]);
  expect(searchExtDefaultSettingsView(extView, 'note').map(item => item.item.name)).toEqual([
    'story',
  ]);
  expect(sortExtDefaultSettingsView(extView, 'modified').map(item => item.item.name)).toEqual([
    'fun',
    'story',
    'coc7',
  ]);
  expect(sortExtDefaultSettingsView(extView, 'name').map(item => item.item.name)).toEqual([
    'coc7',
    'fun',
    'story',
  ]);
  expect(sortExtDefaultSettingsView(extView, 'disabled-count').map(item => item.item.name)).toEqual(
    ['coc7', 'story', 'fun']
  );
  expect(getExtDefaultSettingPage(extView, 3, 2)).toEqual({
    page: 2,
    pageCount: 2,
    items: [extView[2]],
    total: 3,
  });
});

it('基础设置只提交实际变更的字段', () => {
  const initial = normalizeBaseSettingValue({
    commandPrefix: ['.'],
    diceMasters: ['QQ:1'],
    noticeIds: [],
    extDefaultSettings: [],
    masterUnlockCode: 'code',
    uiPassword: '------',
    mailEnable: false,
    mailFrom: '',
    mailPassword: '',
    mailSmtp: '',
    trustOnlyMode: false,
    botExtFreeSwitch: false,
    QQEnablePoke: false,
    officialQQFileSendBase64: false,
    officialQQUseMarkdown: false,
    officialQQUseNameCard: false,
    officialQQUseNameCardTemplate: '<font color="#FF0000" size="1">{card}</font>',
    textCmdTrustOnly: false,
    ignoreUnaddressedBotCmd: false,
    aliveNoticeEnable: false,
    aliveNoticeValue: '',
    logSizeNoticeEnable: false,
    logSizeNoticeCount: 500,
    playerNameWrapEnable: true,
    diceRandomMode: 'pcg',
    diceRandomGMEnabled: false,
    onlyLogCommandInGroup: false,
    onlyLogCommandInPrivate: false,
    rateLimitEnabled: false,
    personalReplenishRate: '',
    personalBurst: 1,
    groupReplenishRate: '',
    groupBurst: 1,
    serveAddress: '127.0.0.1:3211',
    trayTooltip: '',
    refuseGroupInvite: false,
    friendAddComment: '',
    workInQQChannel: false,
    QQChannelAutoOn: false,
    QQChannelLogMessage: false,
    defaultCocRuleIndex: '0',
    maxCocCardGen: '5',
    maxExecuteTime: '12',
    messageDelayRangeStart: 0,
    messageDelayRangeEnd: 0,
    quitInactiveThreshold: 0,
    quitInactiveBatchSize: 0,
    quitInactiveBatchWait: 0,
  } satisfies BaseSettingValueResp);

  const current = structuredClone(initial);
  current.masterUnlockCode = 'refreshed-readonly-code';
  current.QQEnablePoke = true;
  current.commandPrefix = ['.', '!'];
  current.diceRandomGMEnabled = true;

  expect(buildBaseSettingPatch(current, initial)).toEqual({
    QQEnablePoke: true,
    commandPrefix: ['.', '!'],
    diceRandomGMEnabled: true,
  });
});
