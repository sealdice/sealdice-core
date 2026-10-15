import {
  appendRealtimeToolTestMessage,
  appendPendingToolTestMessages,
  appendSelfToolTestMessage,
  buildToolTestCommandOptions,
  createInitialToolTestMessages,
  normalizeToolTestSplitOptions,
  avatarDataUrl,
} from './model.js';
import { expect, it } from 'vitest';

it('指令测试消息追加与上下文转换', async () => {
  const privateSeed = createInitialToolTestMessages('private');
  expect(privateSeed.length).toBe(2);
  expect(privateSeed[0]?.kind).toBe('message');
  expect(privateSeed[0]?.mode).toBe('private');
  expect(privateSeed[0]?.senderName).toBe('海豹核心');
  expect(privateSeed[0]?.isBot).toBe(true);
  expect(privateSeed[1]?.kind).toBe('tip');

  const groupSeed = createInitialToolTestMessages('group');
  expect(groupSeed.length).toBe(2);
  expect(groupSeed[0]?.mode).toBe('group');
  expect(groupSeed[0]?.content.includes('群聊窗口')).toBe(true);

  const withSelf = appendSelfToolTestMessage(privateSeed, {
    text: ' .ping ',
    mode: 'private',
    timestamp: 123456,
  });
  expect(withSelf.length).toBe(3);
  expect(withSelf[withSelf.length - 1]?.self).toBe(true);
  expect(withSelf[withSelf.length - 1]?.content).toBe('.ping');

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
  expect(withPending.length).toBe(5);
  expect(withPending[withPending.length - 1]?.self).toBe(false);
  expect(withPending[withPending.length - 1]?.isBot).toBe(true);
  expect(withPending[withPending.length - 1]?.content).toBe('机器人回复');

  expect(normalizeToolTestSplitOptions(undefined)).toEqual({
    defaultKey: 'qq',
    options: [
      { key: 'short', label: '短分段 300', messageSplitLen: 300 },
      { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
      { key: 'unlimited', label: '无限', messageSplitLen: 0 },
    ],
  });

  expect(
    normalizeToolTestSplitOptions({
      defaultKey: 'short',
      options: [
        { key: 'short', label: '短分段 300', messageSplitLen: 300 },
        { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
      ],
    })
  ).toEqual({
    defaultKey: 'short',
    options: [
      { key: 'short', label: '短分段 300', messageSplitLen: 300 },
      { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
    ],
  });

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
  expect(withRealtime[0]?.self).toBe(true);
  expect(withRealtime[0]?.senderRole).toBe('owner');
  expect(avatarDataUrl('owner', '群主').startsWith('data:image/svg+xml')).toBe(true);
});

it('builds command options with optional description and source metadata', () => {
  const metadataCommands = [
    { name: 'reply', description: '.reply on/off // 控制自动回复', source: '核心' },
    { name: 'roll', source: '测试扩展' },
  ];

  const result = buildToolTestCommandOptions(metadataCommands, '.r');
  if (
    JSON.stringify(result) !==
    JSON.stringify([
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
    ])
  ) {
    throw new Error(`unexpected command options: ${JSON.stringify(result)}`);
  }
});
