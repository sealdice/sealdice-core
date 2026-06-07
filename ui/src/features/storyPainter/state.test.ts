import type { StoryPainterLogItem } from './types';
import { defaultStoryPainterOptions } from './types';
import {
  buildStoryPainterChars,
  buildStoryPainterPreviewItems,
  deleteStoryPainterChar,
  isStoryPainterHidden,
  renameStoryPainterChar,
} from './state';
import { normalizeStoryPainterMessage } from './formatters';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const items: StoryPainterLogItem[] = [
  {
    id: 1,
    nickname: 'Alice',
    IMUserId: '1001',
    time: 100,
    message: 'hello [CQ:at,qq=1002]',
    isDice: false,
    commandId: 0,
  },
  {
    id: 2,
    nickname: 'Seal',
    IMUserId: 'bot',
    time: 101,
    message: '<Alice>掷出了 D20=1',
    isDice: true,
    commandId: 1,
  },
  {
    id: 3,
    nickname: 'obWatcher',
    IMUserId: '1003',
    time: 102,
    message: 'hidden',
    isDice: false,
    commandId: 0,
  },
];

const chars = buildStoryPainterChars(items, new Map([['Alice', '#123456']]));
assertEqual(chars.length, 3);
assertDeepEqual(chars.map((item) => item.role), ['角色', '骰子', '隐藏']);
assertEqual(chars[0]?.color, '#123456');
assertEqual(isStoryPainterHidden(items[2]!, chars), true);

const options = defaultStoryPainterOptions();
const normalized = normalizeStoryPainterMessage(items[0]!, chars, options, false);
assertEqual(normalized, 'hello @1002');

const preview = buildStoryPainterPreviewItems(items, chars, options, (item) =>
  normalizeStoryPainterMessage(item, chars, options, false),
);
assertEqual(preview.length, 2);
assertDeepEqual(preview.map((item) => item.index), [0, 1]);

const renamed = renameStoryPainterChar(items, chars[0]!, 'Alicia');
assertEqual(renamed[0]?.nickname, 'Alicia');
assertEqual(renamed[1]?.message, '<Alicia>掷出了 D20=1');

const deleted = deleteStoryPainterChar(items, chars[0]!);
assertDeepEqual(deleted.map((item) => item.nickname), ['Seal', 'obWatcher']);
