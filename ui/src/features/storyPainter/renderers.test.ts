import { defaultStoryPainterOptions, type StoryPainterLogItem } from './types';
import { renderForumText, renderTrgText } from './renderers';
import { expect, it } from 'vitest';

it('论坛代码保留换行与纯文本，回声工坊输出掷骰指令', () => {
  const options = defaultStoryPainterOptions();
  options.timeHide = true;

  const chars = [{ name: 'Alice', IMUserId: '1001', role: '角色' as const, color: '#db2777' }];

  const item: StoryPainterLogItem = {
    id: 1,
    nickname: 'Alice',
    IMUserId: '1001',
    time: 10,
    message: 'first\nsecond <tag>',
    isDice: false,
    commandId: 0,
  };

  const forum = renderForumText(
    item,
    chars,
    options,
    {
      bbsUseSpaceWithMultiLine: true,
      bbsUseColorName: false,
    },
    '#db2777'
  );

  expect(forum).toContain('\n\u2002<Alice>second <tag>');
  expect(forum).not.toContain('<span');
  expect(forum).not.toContain('<br');
  expect(forum).not.toContain('&lt;tag&gt;');

  const trg = renderTrgText(
    {
      ...item,
      message: 'roll',
      isDice: true,
      commandInfo: {
        cmd: 'roll',
        pcName: 'Alice',
        items: [{ expr: 'D100', result: 30 }],
      },
    },
    chars,
    options,
    true
  );

  expect(trg).toContain('# [Alice]:roll');
  expect(trg).toContain('\n<dice>:(Alice的D100,100,NA,30)');
});
