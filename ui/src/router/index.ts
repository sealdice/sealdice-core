import { createRouter, createWebHashHistory, type RouteLocationNormalized, type RouteRecordRaw } from 'vue-router';
import { handleHotUpdate, routes } from 'vue-router/auto-routes';
import { setupRouterProgress } from './progress';
import { routeMeta } from './routeMeta';
import { setupUnsavedChangesGuard } from '@/features/unsavedChanges';
import './types';

// vue-router/auto-routes 只负责从 pages/ 生成路径。项目级的标题、布局和
// 少量参数适配集中在这里合并，避免把元信息散落在每个页面文件里。
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

    // 文件路由会正确生成动态 path，但页面组件需要稳定的 string prop。
    // 在这里统一转换，页面就不用关心 route.params 的 string/string[] 分支。
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
  // 后端以内嵌静态资源形式分发 UI，Hash history 不要求后端为每个前端路由配 fallback。
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
setupUnsavedChangesGuard(router);

export default router;
