import type { NavigationItem } from './types';

// 侧边栏菜单是独立于文件路由的产品导航模型。
// 不是所有路由都一定出现在菜单里，菜单也可以挂动态子项（如自定义文案分类）。
// 路由标题、布局和面包屑也从这里派生，避免多个文件维护同一份页面语义。
export const appNavigation: NavigationItem[] = [
  {
    label: '主页',
    title: '主页',
    path: '/',
    icon: 'home',
    layout: 'default',
  },
  {
    label: '账号设置',
    title: '账号设置',
    path: '/connect',
    icon: 'connection',
    layout: 'default',
  },
  {
    label: '自定义文案',
    title: '自定义文案',
    icon: 'setting',
    layout: 'default',
    dynamicChildren: 'customTextCategories',
  },
  {
    label: '扩展功能',
    icon: 'edit',
    children: [
      {
        label: '自定义回复',
        title: '自定义回复',
        path: '/mod/reply',
        icon: 'reply',
        layout: 'wide',
      },
      {
        label: '牌堆管理',
        title: '牌堆管理',
        path: '/mod/deck',
        icon: 'deck',
        layout: 'wide',
      },
      {
        label: '跑团日志',
        title: '跑团日志',
        path: '/mod/story',
        icon: 'story',
        layout: 'wide',
      },
      {
        label: 'JS 扩展',
        title: 'JS 扩展',
        path: '/mod/js',
        icon: 'js',
        layout: 'wide',
      },
      {
        label: '帮助文档',
        title: '帮助文档',
        path: '/mod/helpdoc',
        icon: 'helpdoc',
        layout: 'wide',
      },
      {
        label: '拦截管理',
        title: '拦截管理',
        path: '/mod/censor',
        icon: 'censor',
        layout: 'wide',
      },
    ],
  },
  {
    label: '综合设置',
    icon: 'operation',
    children: [
      {
        label: '基本设置',
        title: '基本设置',
        path: '/misc/base-setting',
        icon: 'base-setting',
        layout: 'wide',
      },
      {
        label: '群组管理',
        title: '群组管理',
        path: '/misc/group',
        icon: 'group',
        layout: 'default',
      },
      {
        label: '黑白名单',
        title: '黑白名单',
        path: '/misc/ban',
        icon: 'ban',
        layout: 'default',
      },
      {
        label: '公骰设置',
        title: '公骰设置',
        path: '/misc/dice-public',
        icon: 'dice',
        layout: 'wide',
      },
      {
        label: '备份',
        title: '备份',
        path: '/misc/backup',
        icon: 'backup',
        layout: 'wide',
      },
      {
        label: '高级设置',
        title: '高级设置',
        path: '/misc/advanced-setting',
        icon: 'advanced-setting',
        layout: 'default',
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
        title: '指令测试',
        path: '/tool/test',
        icon: 'test',
        layout: 'default',
      },
      {
        label: '资源管理',
        title: '资源管理',
        path: '/tool/resource',
        icon: 'resource',
        layout: 'wide',
      },
    ],
  },
  {
    label: '关于',
    title: '关于',
    path: '/about',
    icon: 'star',
    layout: 'wide',
  },
];
