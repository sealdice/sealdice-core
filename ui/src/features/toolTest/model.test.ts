import {
  appendRealtimeToolTestMessage,
  appendPendingToolTestMessages,
  appendSelfToolTestMessage,
  buildToolTestCommandOptions,
  createInitialToolTestMessages,
  normalizeToolTestSplitOptions,
  avatarDataUrl,
} from './model.js';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    const normalize = (value: unknown): string => {
      if (Array.isArray(value)) {
        return `[${value.map(normalize).join(',')}]`;
      }
      if (value && typeof value === 'object') {
        const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
          a.localeCompare(b)
        );
        return `{${entries.map(([key, entry]) => `${key}:${normalize(entry)}`).join(',')}}`;
      }
      return JSON.stringify(value);
    };
    if (normalize(actual) !== normalize(expected)) {
      throw new Error(`expected ${normalize(expected)}, got ${normalize(actual)}`);
    }
  };

  const privateSeed = createInitialToolTestMessages('private');
  assertEqual(privateSeed.length, 2);
  assertEqual(privateSeed[0]?.kind, 'message');
  assertEqual(privateSeed[0]?.mode, 'private');
  assertEqual(privateSeed[0]?.senderName, '海豹核心');
  assertEqual(privateSeed[0]?.isBot, true);
  assertEqual(privateSeed[1]?.kind, 'tip');

  const groupSeed = createInitialToolTestMessages('group');
  assertEqual(groupSeed.length, 2);
  assertEqual(groupSeed[0]?.mode, 'group');
  assertEqual(groupSeed[0]?.content.includes('群聊窗口'), true);

  const withSelf = appendSelfToolTestMessage(privateSeed, {
    text: ' .ping ',
    mode: 'private',
    timestamp: 123456,
  });
  assertEqual(withSelf.length, 3);
  assertEqual(withSelf[withSelf.length - 1]?.self, true);
  assertEqual(withSelf[withSelf.length - 1]?.content, '.ping');

  const withPending = appendPendingToolTestMessages(
    withSelf,
    [
      { uid: 'UI:1001', message: 'ignored', messageType: 'private' },
      { uid: 'UI:1002', message: '群消息', messageType: 'group' },
      { uid: 'UI:1001', message: '机器人回复', messageType: 'private' },
    ],
    'private',
    223344
  );
  assertEqual(withPending.length, 5);
  assertEqual(withPending[withPending.length - 1]?.self, false);
  assertEqual(withPending[withPending.length - 1]?.isBot, true);
  assertEqual(withPending[withPending.length - 1]?.content, '机器人回复');

  assertDeepEqual(normalizeToolTestSplitOptions(undefined), {
    defaultKey: 'qq',
    options: [
      { key: 'short', label: '短分段 300', messageSplitLen: 300 },
      { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
      { key: 'unlimited', label: '无限', messageSplitLen: 0 },
    ],
  });

  assertDeepEqual(
    normalizeToolTestSplitOptions({
      defaultKey: 'short',
      options: [
        { key: 'short', label: '短分段 300', messageSplitLen: 300 },
        { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
      ],
    }),
    {
      defaultKey: 'short',
      options: [
        { key: 'short', label: '短分段 300', messageSplitLen: 300 },
        { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
      ],
    }
  );

  const context = {
    mode: 'group' as const,
    conversationId: 'UI-Group:2001',
    groupId: 'UI-Group:2001',
    groupName: '测试群',
    currentSenderId: 'UI:1002',
    members: [],
    botName: '海豹核心',
    botAvatarKey: 'seal',
    commandPrefix: ['.', '!'],
  };
  const withRealtime = appendRealtimeToolTestMessage(
    [],
    {
      id: 'event-1',
      messageType: 'group',
      conversationId: 'UI-Group:2001',
      sender: { userId: 'UI:1002', nickname: '群主' },
      senderRole: 'owner',
      avatarKey: 'owner',
      isBot: false,
      direction: 'incoming',
      rawMessage: '.r 1',
      segments: [{ type: 'text', text: '.r 1' }],
      timestamp: 10,
    },
    context
  );
  assertEqual(withRealtime[0]?.self, true);
  assertEqual(withRealtime[0]?.senderRole, 'owner');
  assertEqual(avatarDataUrl('owner', '群主').startsWith('data:image/svg+xml'), true);
});

it('builds command options with optional description and source metadata', () => {
  const metadataCommands = [
    { name: 'reply', description: '.reply on/off // 控制自动回复', source: '核心' },
    { name: 'roll', source: '测试扩展' },
  ];

  const result = buildToolTestCommandOptions(metadataCommands, '.r');
  if (JSON.stringify(result) !== JSON.stringify([
    {
      label: '.reply',
      value: '.reply',
      description: '.reply on/off // 控制自动回复',
      source: '核心',
    },
    {
      label: '.roll',
      value: '.roll',
      source: '测试扩展',
    },
  ])) {
    throw new Error(`unexpected command options: ${JSON.stringify(result)}`);
  }
});
