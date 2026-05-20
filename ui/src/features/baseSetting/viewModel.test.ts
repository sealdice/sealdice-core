import {
  buildBaseSettingPatch,
  buildBaseSettingSearchIndex,
  normalizeBaseSettingSchema,
  normalizeBaseSettingValue,
  searchBaseSettingFields,
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
} as any);

const index = buildBaseSettingSearchIndex(schema);
assertEqual(index.length, 1);
assertEqual(searchBaseSettingFields(index, '戳一戳')[0]?.fieldId, 'qq-enable-poke');
assertEqual(searchBaseSettingFields(index, '平台')[0]?.tabId, 'platform-special');

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
} as any);

const current = structuredClone(initial);
current.QQEnablePoke = true;
current.commandPrefix = ['.', '!'];

assertDeepEqual(buildBaseSettingPatch(current, initial), {
  QQEnablePoke: true,
  commandPrefix: ['.', '!'],
});
