import type { StoryPainterLogItem } from './types';
import { defaultStoryPainterOptions } from './types';
import {
  buildStoryPainterChars,
  buildStoryPainterPreviewItems,
  deleteStoryPainterChar,
  renameStoryPainterChar,
} from './state';
import { normalizeStoryPainterMessage } from './formatters';
import { expect, it } from 'vitest';

it('日志阅读状态与选项更新', async () => {
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
  expect(chars.length).toBe(3);
  expect(chars.map(item => item.role)).toEqual(['角色', '骰子', '隐藏']);
  expect(chars[0]?.color).toBe('#123456');

  const options = defaultStoryPainterOptions();
  const normalized = normalizeStoryPainterMessage(items[0]!, chars, options, false);
  expect(normalized).toBe('hello @1002');

  const preview = buildStoryPainterPreviewItems(items, chars, item =>
    normalizeStoryPainterMessage(item, chars, options, false)
  );
  expect(preview.length).toBe(2);
  expect(preview.map(item => item.index)).toEqual([0, 1]);

  const renamed = renameStoryPainterChar(items, chars[0]!, 'Alicia');
  expect(renamed[0]?.nickname).toBe('Alicia');
  expect(renamed[1]?.message).toBe('<Alicia>掷出了 D20=1');

  const deleted = deleteStoryPainterChar(items, chars[0]!);
  expect(deleted.map(item => item.nickname)).toEqual(['Seal', 'obWatcher']);
});
