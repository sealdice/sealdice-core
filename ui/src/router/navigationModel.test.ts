import {
  addSearchHistory,
  buildBreadcrumbItems,
  buildNavigationTree,
  buildRouteMeta,
  flattenNavigationItems,
  getNavigationExpandedKeys,
  matchesNavigationSearch,
  removeSearchHistoryItem,
  resolveMenuNavigationTarget,
} from './navigationModel.ts';
import type { AppLayoutName, NavigationItem } from './types.ts';
import { expect, it } from 'vitest';

it('从导航模型派生路由与动态菜单', async () => {
  const sourceItems: NavigationItem[] = [
    {
      label: '运行概览',
      path: '/',
      icon: 'home',
      title: '运行概览',
      layout: 'default',
    },
    {
      label: '自定义文案',
      icon: 'edit',
      title: '自定义文案',
      layout: 'default',
      dynamicChildren: 'customTextCategories',
      children: [],
    },
    {
      label: '综合设置',
      icon: 'operation',
      children: [
        {
          label: '基本设置',
          path: '/misc/base-setting',
          title: '基本设置',
          layout: 'default',
        },
        {
          label: '高级设置',
          path: '/misc/advanced-setting',
          title: '高级设置',
          layout: 'default',
          requiresAdvancedConfig: true,
        },
      ],
    },
    {
      label: '扩展功能',
      icon: 'edit',
      children: [
        {
          label: '自定义回复',
          path: '/mod/reply',
          title: '自定义回复',
          layout: 'workspace',
        },
      ],
    },
    {
      label: '隐藏项',
      path: '/hidden',
      hidden: true,
    },
  ];

  const lockedTree = buildNavigationTree(sourceItems, {
    advancedConfigEnabled: false,
    customTextCategories: ['默认', '骰子核心'],
  });

  const lockedLeaves = flattenNavigationItems(lockedTree);
  expect(lockedLeaves.map(item => item.path)).toEqual([
    '/',
    '/custom-text/默认',
    '/custom-text/骰子核心',
    '/misc/base-setting',
    '/mod/reply',
  ]);

  const unlockedLeaves = flattenNavigationItems(
    buildNavigationTree(sourceItems, {
      advancedConfigEnabled: true,
      customTextCategories: ['默认'],
    })
  );
  expect(unlockedLeaves.map(item => item.path)).toEqual([
    '/',
    '/custom-text/默认',
    '/misc/base-setting',
    '/misc/advanced-setting',
    '/mod/reply',
  ]);
  expect(getNavigationExpandedKeys(lockedTree, '/misc/base-setting')).toEqual(['综合设置']);
  expect(getNavigationExpandedKeys(lockedTree, '/custom-text/默认')).toEqual(['自定义文案']);
  expect(getNavigationExpandedKeys(lockedTree, '/')).toEqual([]);

  const baseSetting = lockedLeaves.find(item => item.path === '/misc/base-setting');
  if (!baseSetting) throw new Error('expected base setting search item');
  expect(matchesNavigationSearch(baseSetting, '基本')).toBe(true);
  expect(matchesNavigationSearch(baseSetting, 'jbsz')).toBe(true);
  expect(matchesNavigationSearch(baseSetting, 'jibenshezhi')).toBe(true);
  expect(matchesNavigationSearch(baseSetting, 'does-not-exist')).toBe(false);

  const history = addSearchHistory([], baseSetting);
  expect(history).toEqual([baseSetting]);
  expect(addSearchHistory(history, baseSetting)).toEqual([baseSetting]);
  expect(removeSearchHistoryItem(history, baseSetting.path)).toEqual([]);

  const routeMeta = buildRouteMeta(sourceItems);
  expect(routeMeta['/']).toEqual({ title: '运行概览', layout: 'default' satisfies AppLayoutName });
  expect(routeMeta['/custom-text/:category']).toEqual({
    title: '自定义文案',
    layout: 'default' satisfies AppLayoutName,
  });
  expect(routeMeta['/misc/base-setting']).toEqual({
    title: '基本设置',
    layout: 'default' satisfies AppLayoutName,
  });
  expect(routeMeta['/mod/reply']).toEqual({
    title: '自定义回复',
    layout: 'workspace' satisfies AppLayoutName,
  });
  expect(routeMeta['/hidden']).toBe(undefined);

  expect(buildBreadcrumbItems(sourceItems, '/')).toEqual([{ label: '运行概览' }]);
  expect(buildBreadcrumbItems(sourceItems, '/misc/base-setting')).toEqual([
    { label: '综合设置' },
    { label: '基本设置' },
  ]);
  expect(buildBreadcrumbItems(sourceItems, '/custom-text/默认')).toEqual([
    { label: '自定义文案' },
    { label: '默认' },
  ]);
});

it('resolveMenuNavigationTarget 只在折叠态兜底跳转', () => {
  // 折叠态：点击路由项返回目标路径
  expect(resolveMenuNavigationTarget('/mod/story', '/', true)).toBe('/mod/story');
  // 展开态：交给菜单项内的 RouterLink，不重复导航
  expect(resolveMenuNavigationTarget('/mod/story', '/', false)).toBe(null);
  // 当前页不重复跳转
  expect(resolveMenuNavigationTarget('/mod/story', '/mod/story', true)).toBe(null);
  // 分组标题等非路由 key 不跳转
  expect(resolveMenuNavigationTarget('综合设置', '/', true)).toBe(null);
  // 当前路径带 URL 编码时按解码后比较
  expect(
    resolveMenuNavigationTarget('/custom-text/默认', '/custom-text/%E9%BB%98%E8%AE%A4', true)
  ).toBe(null);
});
