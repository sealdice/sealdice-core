import {
  addSearchHistory,
  buildNavigationTree,
  flattenNavigationItems,
  getNavigationExpandedKeys,
  matchesNavigationSearch,
  removeSearchHistoryItem,
} from './navigationModel.ts';
import type { NavigationItem } from './types.ts';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const sourceItems: NavigationItem[] = [
  {
    label: '主页',
    path: '/',
    icon: 'home',
  },
  {
    label: '自定义文案',
    icon: 'edit',
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
      },
      {
        label: '高级设置',
        path: '/misc/advanced-setting',
        requiresAdvancedConfig: true,
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
assertDeepEqual(
  lockedLeaves.map(item => item.path),
  ['/', '/custom-text/默认', '/custom-text/骰子核心', '/misc/base-setting'],
);

const unlockedLeaves = flattenNavigationItems(
  buildNavigationTree(sourceItems, {
    advancedConfigEnabled: true,
    customTextCategories: ['默认'],
  }),
);
assertDeepEqual(
  unlockedLeaves.map(item => item.path),
  ['/', '/custom-text/默认', '/misc/base-setting', '/misc/advanced-setting'],
);
assertDeepEqual(getNavigationExpandedKeys(lockedTree, '/misc/base-setting'), ['综合设置']);
assertDeepEqual(getNavigationExpandedKeys(lockedTree, '/custom-text/默认'), ['自定义文案']);
assertDeepEqual(getNavigationExpandedKeys(lockedTree, '/'), []);

const baseSetting = lockedLeaves.find(item => item.path === '/misc/base-setting');
if (!baseSetting) throw new Error('expected base setting search item');
assertEqual(matchesNavigationSearch(baseSetting, '基本'), true);
assertEqual(matchesNavigationSearch(baseSetting, 'jbsz'), true);
assertEqual(matchesNavigationSearch(baseSetting, 'jibenshezhi'), true);
assertEqual(matchesNavigationSearch(baseSetting, 'does-not-exist'), false);

const history = addSearchHistory([], baseSetting);
assertDeepEqual(history, [baseSetting]);
assertDeepEqual(addSearchHistory(history, baseSetting), [baseSetting]);
assertDeepEqual(removeSearchHistoryItem(history, baseSetting.path), []);
