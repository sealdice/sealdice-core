import { defaultStoryPainterOptions, type StoryPainterLogItem } from './types';
import { renderForumText, renderTrgText } from './renderers';

const assertIncludes = (actual: string, expected: string) => {
  if (!actual.includes(expected)) {
    throw new Error(`expected ${JSON.stringify(actual)} to include ${JSON.stringify(expected)}`);
  }
};

const assertNotIncludes = (actual: string, expected: string) => {
  if (actual.includes(expected)) {
    throw new Error(`expected ${JSON.stringify(actual)} not to include ${JSON.stringify(expected)}`);
  }
};

const options = defaultStoryPainterOptions();
options.timeHide = true;

const chars = [
  { name: 'Alice', IMUserId: '1001', role: '角色' as const, color: '#db2777' },
];

const item: StoryPainterLogItem = {
  id: 1,
  nickname: 'Alice',
  IMUserId: '1001',
  time: 10,
  message: 'first\nsecond <tag>',
  isDice: false,
  commandId: 0,
};

const forum = renderForumText(item, chars, options, {
  bbsUseSpaceWithMultiLine: true,
  bbsUseColorName: false,
}, '#db2777');

assertIncludes(forum, '\n\u2002<Alice>second <tag>');
assertNotIncludes(forum, '<span');
assertNotIncludes(forum, '<br');
assertNotIncludes(forum, '&lt;tag&gt;');

const trg = renderTrgText({
  ...item,
  message: 'roll',
  isDice: true,
  commandInfo: {
    cmd: 'roll',
    pcName: 'Alice',
    items: [{ expr: 'D100', result: 30 }],
  },
}, chars, options, true);

assertIncludes(trg, '# [Alice]:roll');
assertIncludes(trg, '\n<dice>:(Alice的D100,100,NA,30)');
