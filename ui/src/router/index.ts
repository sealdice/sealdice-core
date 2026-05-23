import { createRouter, createWebHashHistory } from 'vue-router';
import { handleHotUpdate, routes } from 'vue-router/auto-routes';
import { resolveHashHistoryBase } from './historyBase';
import { setupRouterProgress } from './progress';
import { routeMeta } from './routeMeta';
import { withRouteMeta } from './routeRecords';
import { setupUnsavedChangesGuard } from '@/features/unsavedChanges';
import './types';

const appRoutes = withRouteMeta(routes, routeMeta);

const router = createRouter({
  // 后端以内嵌静态资源形式分发 UI，Hash history 不要求后端为每个前端路由配 fallback。
  history: createWebHashHistory(resolveHashHistoryBase(import.meta.env.BASE_URL)),
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
