import type { RouteRecordRaw } from 'vue-router';

// routeMeta 是页面标题和布局选择的单一事实源。
// 新增 pages/* 后如果不在这里登记，页面仍能访问，但会落到 default layout
// 且面包屑/菜单标题缺少业务语义。
export const routeMeta: Record<string, RouteRecordRaw['meta']> = {
  '/': { layout: 'default', title: '主页' },
  '/connect': { layout: 'default', title: '账号设置' },
  '/custom-text/:category': { layout: 'default', title: '自定义文案' },
  '/mod/reply': { layout: 'wide', title: '自定义回复' },
  '/mod/deck': { layout: 'wide', title: '牌堆管理' },
  '/mod/story': { layout: 'wide', title: '跑团日志' },
  '/mod/js': { layout: 'wide', title: 'JS 扩展' },
  '/mod/helpdoc': { layout: 'default', title: '帮助文档' },
  '/mod/censor': { layout: 'default', title: '拦截管理' },
  '/misc/base-setting': { layout: 'default', title: '基本设置' },
  '/misc/group': { layout: 'default', title: '群组管理' },
  '/misc/ban': { layout: 'default', title: '黑白名单' },
  '/misc/dice-public': { layout: 'default', title: '公骰设置' },
  '/misc/backup': { layout: 'default', title: '备份' },
  '/misc/advanced-setting': { layout: 'default', title: '高级设置' },
  '/tool/test': { layout: 'default', title: '指令测试' },
  '/tool/resource': { layout: 'default', title: '资源管理' },
  '/about': { layout: 'default', title: '关于' },
};
