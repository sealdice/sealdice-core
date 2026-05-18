import { createRouter, createWebHashHistory, type RouteLocationNormalized, type RouteRecordRaw } from 'vue-router';
import { handleHotUpdate, routes } from 'vue-router/auto-routes';
import { setupRouterProgress } from './progress';
import './types';

const routeMeta: Record<string, RouteRecordRaw['meta']> = {
  '/': { layout: 'default', title: '主页' },
  '/connect': { layout: 'default', title: '账号设置' },
  '/custom-text/:category': { layout: 'default', title: '自定义文案' },
  '/mod/reply': { layout: 'default', title: '自定义回复' },
  '/mod/deck': { layout: 'default', title: '牌堆管理' },
  '/mod/story': { layout: 'default', title: '跑团日志' },
  '/mod/js': { layout: 'default', title: 'JS 扩展' },
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

function withRouteMeta(records: readonly RouteRecordRaw[]): RouteRecordRaw[] {
  return records.map(record => {
    const baseRecord = record as RouteRecordRaw;
    const nextRecord = {
      ...baseRecord,
      meta: {
        ...baseRecord.meta,
        ...routeMeta[baseRecord.path],
      },
      children: baseRecord.children ? withRouteMeta(baseRecord.children) : baseRecord.children,
    } as RouteRecordRaw;

    if (baseRecord.path === '/custom-text/:category') {
      nextRecord.props = (route: RouteLocationNormalized) => ({
        category: String((route.params as Record<string, string | string[] | undefined>)['category'] ?? ''),
      });
    }

    return nextRecord;
  });
}

const appRoutes = withRouteMeta(routes);

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    ...appRoutes,
    { path: '/home', redirect: '/' },
    { path: '/signin', redirect: '/' },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
});

if (import.meta.hot) {
  handleHotUpdate(router);
}

setupRouterProgress(router);

export default router;
