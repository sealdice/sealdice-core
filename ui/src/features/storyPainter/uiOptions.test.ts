import { storyPainterOptionList } from './uiOptions';

const assert = (condition: boolean, message: string) => {
  if (!condition) throw new Error(message);
};

assert(storyPainterOptionList.length >= 7, 'story painter options should keep original option descriptions');
for (const item of storyPainterOptionList) {
  assert(item.label.trim().length > 0, `option ${String(item.key)} is missing label`);
  assert(item.desc.trim().length > 0, `option ${String(item.key)} is missing description`);
}

assert(
  storyPainterOptionList.some(item => item.key === 'commandHide' && item.desc.includes('正常显示指令结果')),
  'commandHide description should match original behavior',
);
