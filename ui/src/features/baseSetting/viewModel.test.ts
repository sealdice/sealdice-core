import {
  buildBaseSettingPatch,
  buildBaseSettingSearchIndex,
  buildExtDefaultSettingsView,
  buildBaseSettingStringListOptions,
  filterExtDefaultSettingsView,
  getExtDefaultSettingPage,
  getBaseSettingFieldFeedback,
  getBaseSettingFieldLayout,
  getExtDefaultSettingModifiedCount,
  isBaseSettingFieldBottomMounted,
  isBaseSettingGroupWide,
  normalizeBaseSettingSchema,
  normalizeBaseSettingValue,
  searchBaseSettingFields,
  searchExtDefaultSettingsView,
  sortExtDefaultSettingsView,
} from './viewModel.js';
import type {
  BaseSettingExtDefaultSettingItem,
  BaseSettingFieldSchema,
  BaseSettingSchemaResp,
  BaseSettingValueResp,
} from '@/api';

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

const schema = normalizeBaseSettingSchema({
  tabs: [
    {
      id: 'platform-special',
      title: '平台特殊配置',
      groups: [
        {
          id: 'qq-channel',
          title: 'QQ 频道设置',
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
} satisfies BaseSettingSchemaResp);

const index = buildBaseSettingSearchIndex(schema);
assertEqual(index.length, 1);
assertEqual(searchBaseSettingFields(index, '戳一戳')[0]?.fieldId, 'qq-enable-poke');
assertEqual(searchBaseSettingFields(index, '平台')[0]?.tabId, 'platform-special');
assertEqual(isBaseSettingGroupWide('ext-default-settings'), true);
assertEqual(isBaseSettingGroupWide('upgrade'), true);
assertEqual(isBaseSettingGroupWide('mail'), false);
assertEqual(isBaseSettingFieldBottomMounted({ kind: 'string-list' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), true);
assertEqual(isBaseSettingFieldBottomMounted({ kind: 'boolean' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), false);
assertEqual(getBaseSettingFieldLayout({ kind: 'boolean' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), 'inline');
assertEqual(getBaseSettingFieldLayout({ kind: 'text' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), 'auto');
assertEqual(getBaseSettingFieldLayout({ kind: 'string-list' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), 'stacked');
assertEqual(getBaseSettingFieldLayout({ kind: 'upload' } satisfies Pick<BaseSettingFieldSchema, 'kind'>), 'stacked');
assertEqual(getBaseSettingFieldFeedback({ key: 'QQEnablePoke' } satisfies Pick<BaseSettingFieldSchema, 'key' | 'hint'>), '启用前请确认你使用的 QQ 连接方式支持该功能，若不支持请关闭该功能来避免日志中出现相关报错。');
assertEqual(getBaseSettingFieldFeedback({ hint: '用于发送通知' } satisfies Pick<BaseSettingFieldSchema, 'key' | 'hint'>), '用于发送通知');
assertDeepEqual(buildBaseSettingStringListOptions(['.', '!', '.', '', ' QQ:1 ']), [
  { label: '.', value: '.' },
  { label: '!', value: '!' },
  { label: 'QQ:1', value: 'QQ:1' },
]);

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
assertEqual(extView.length, 3);
assertEqual(extView[0]?.dirty, false);
assertEqual(extView[1]?.dirty, true);
assertEqual(extView[1]?.autoActiveDirty, true);
assertEqual(extView[2]?.disabledCommandDirty, true);
assertDeepEqual(extView[2]?.changedCommands, ['note']);
assertEqual(getExtDefaultSettingModifiedCount(extView), 2);
assertDeepEqual(filterExtDefaultSettingsView(extView, 'modified').map(item => item.item.name), ['fun', 'story']);
assertDeepEqual(searchExtDefaultSettingsView(extView, 'note').map(item => item.item.name), ['story']);
assertDeepEqual(sortExtDefaultSettingsView(extView, 'modified').map(item => item.item.name), ['fun', 'story', 'coc7']);
assertDeepEqual(sortExtDefaultSettingsView(extView, 'name').map(item => item.item.name), ['coc7', 'fun', 'story']);
assertDeepEqual(sortExtDefaultSettingsView(extView, 'disabled-count').map(item => item.item.name), ['coc7', 'story', 'fun']);
assertDeepEqual(getExtDefaultSettingPage(extView, 3, 2), {
  page: 2,
  pageCount: 2,
  items: [extView[2]],
  total: 3,
});

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
  textCmdTrustOnly: false,
  ignoreUnaddressedBotCmd: false,
  aliveNoticeEnable: false,
  aliveNoticeValue: '',
  logSizeNoticeEnable: false,
  logSizeNoticeCount: 500,
  playerNameWrapEnable: true,
  onlyLogCommandInGroup: false,
  onlyLogCommandInPrivate: false,
  rateLimitEnabled: false,
  personalReplenishRate: '',
  personalBurst: 1,
  groupReplenishRate: '',
  groupBurst: 1,
  serveAddress: '127.0.0.1:3211',
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
current.QQEnablePoke = true;
current.commandPrefix = ['.', '!'];

assertDeepEqual(buildBaseSettingPatch(current, initial), {
  QQEnablePoke: true,
  commandPrefix: ['.', '!'],
});
