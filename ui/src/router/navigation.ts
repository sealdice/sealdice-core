import type { NavigationItem } from './types';

export const appNavigation: NavigationItem[] = [
  {
    label: '主页',
    path: '/',
    icon: 'home',
  },
  {
    label: '账号设置',
    path: '/connect',
    icon: 'connection',
  },
  {
    label: '自定义文案',
    icon: 'setting',
    dynamicChildren: 'customTextCategories',
  },
  {
    label: '扩展功能',
    icon: 'edit',
    children: [
      {
        label: '自定义回复',
        path: '/mod/reply',
        icon: 'reply',
      },
      {
        label: '牌堆管理',
        path: '/mod/deck',
        icon: 'deck',
      },
      {
        label: '跑团日志',
        path: '/mod/story',
        icon: 'story',
      },
      {
        label: 'JS 扩展',
        path: '/mod/js',
        icon: 'js',
      },
      {
        label: '帮助文档',
        path: '/mod/helpdoc',
        icon: 'helpdoc',
      },
      {
        label: '拦截管理',
        path: '/mod/censor',
        icon: 'censor',
      },
    ],
  },
  {
    label: '综合设置',
    icon: 'operation',
    children: [
      {
        label: '基本设置',
        path: '/misc/base-setting',
        icon: 'base-setting',
      },
      {
        label: '群组管理',
        path: '/misc/group',
        icon: 'group',
      },
      {
        label: '黑白名单',
        path: '/misc/ban',
        icon: 'ban',
      },
      {
        label: '公骰设置',
        path: '/misc/dice-public',
        icon: 'dice',
      },
      {
        label: '备份',
        path: '/misc/backup',
        icon: 'backup',
      },
      {
        label: '高级设置',
        path: '/misc/advanced-setting',
        icon: 'advanced-setting',
        requiresAdvancedConfig: true,
      },
    ],
  },
  {
    label: '辅助工具',
    icon: 'tools',
    children: [
      {
        label: '指令测试',
        path: '/tool/test',
        icon: 'test',
      },
      {
        label: '资源管理',
        path: '/tool/resource',
        icon: 'resource',
      },
    ],
  },
  {
    label: '关于',
    path: '/about',
    icon: 'star',
  },
];
