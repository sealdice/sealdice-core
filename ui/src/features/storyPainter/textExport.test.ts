import { defaultStoryPainterOptions, type StoryPainterLogItem } from './types';
import { collectStoryPainterForumText, collectStoryPainterTrgText } from './textExport';
import { expect, it } from 'vitest';

it('日志导出文本保留内容与顺序', async () => {
  async function* chunks(items: StoryPainterLogItem[][]): AsyncGenerator<StoryPainterLogItem[]> {
    for (const chunk of items) yield chunk;
  }

  const alice1: StoryPainterLogItem = {
    id: 1,
    nickname: 'Alice',
    IMUserId: '1001',
    time: 10,
    message: 'one',
    isDice: false,
    commandId: 0,
  };

  const alice2: StoryPainterLogItem = {
    ...alice1,
    id: 2,
    message: 'two',
  };

  const bob: StoryPainterLogItem = {
    ...alice1,
    id: 3,
    nickname: 'Bob',
    IMUserId: '1002',
    message: 'three',
  };

  const options = defaultStoryPainterOptions();
  const chars = [
    { name: 'Alice', IMUserId: '1001', role: '角色' as const, color: '#db2777' },
    { name: 'Bob', IMUserId: '1002', role: '角色' as const, color: '#0284c7' },
  ];
  const forumOptions = {
    bbsUseSpaceWithMultiLine: false,
    bbsUseColorName: false,
  };

  const pineapple = await collectStoryPainterForumText({
    chunks: chunks([[alice1], [alice2, bob]]),
    chars,
    exportOptions: options,
    forumOptions,
    pineapple: true,
    colorByItem: item => (item.nickname === 'Alice' ? '#db2777' : '#0284c7'),
  });

  expect(pineapple).toBe(
    '[color=silver]<Alice>[/color][color=#db2777] one\ntwo [/color]\n[color=silver]<Bob>[/color][color=#0284c7] three [/color]'
  );

  const trg = await collectStoryPainterTrgText({
    chunks: chunks([[alice1], [bob]]),
    chars,
    exportOptions: options,
    addVoiceMark: true,
  });

  expect(trg).toBe('[Alice]:one{*}\n[Bob]:three{*}');
});
