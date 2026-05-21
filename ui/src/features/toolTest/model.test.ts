import {
  appendPendingToolTestMessages,
  appendSelfToolTestMessage,
  buildToolTestCommandOptions,
  createInitialToolTestMessages,
} from './model.js';

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

const withPending = appendPendingToolTestMessages(withSelf, [
  { uid: 'UI:1001', message: 'ignored', messageType: 'private' },
  { uid: 'UI:1002', message: '群消息', messageType: 'group' },
  { uid: 'UI:1001', message: '机器人回复', messageType: 'private' },
], 'private', 223344);
assertEqual(withPending.length, 5);
assertEqual(withPending[withPending.length - 1]?.self, false);
assertEqual(withPending[withPending.length - 1]?.isBot, true);
assertEqual(withPending[withPending.length - 1]?.content, '机器人回复');

assertDeepEqual(buildToolTestCommandOptions(['reply', 'r', 'roll'], ''), [
  { label: '.reply', value: '.reply' },
  { label: '.r', value: '.r' },
  { label: '.roll', value: '.roll' },
]);
assertDeepEqual(buildToolTestCommandOptions(['reply', 'r', 'roll'], '.r'), [
  { label: '.reply', value: '.reply' },
  { label: '.r', value: '.r' },
  { label: '.roll', value: '.roll' },
]);
assertDeepEqual(buildToolTestCommandOptions(['reply', 'r', 'roll'], '!ro'), [
  { label: '!roll', value: '!roll' },
]);
