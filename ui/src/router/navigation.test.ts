import { expect, it } from 'vitest';
import { buildRouteMeta } from './navigationModel';
import { appNavigation } from './navigation';

const routeMeta = buildRouteMeta(appNavigation);

it.each([
  ['/mod/reply', 'workspace'],
  ['/mod/deck', 'wide'],
  ['/mod/story', 'wide'],
  ['/mod/js', 'wide'],
  ['/mod/package', 'wide'],
  ['/mod/helpdoc', 'wide'],
  ['/mod/censor', 'wide'],
  ['/misc/base-setting', 'default'],
  ['/misc/group', 'wide'],
  ['/misc/ban', 'default'],
  ['/misc/backup', 'default'],
  ['/misc/advanced-setting', 'default'],
  ['/misc/dice-public', 'wide'],
  ['/tool/test', 'workspace'],
  ['/', 'default'],
  ['/connect', 'default'],
  ['/about', 'wide'],
])('uses the product layout for %s', (path, layout) => {
  expect(routeMeta[path!]?.layout).toBe(layout);
});

it('keeps extension navigation in the intended display order', () => {
  expect(
    appNavigation.find(item => item.label === '扩展功能')?.children?.map(item => item.label)
  ).toEqual([
    '自定义回复',
    '扩展包管理',
    'JS 扩展',
    '牌堆管理',
    '跑团日志',
    '帮助文档',
    '拦截管理',
  ]);
});
