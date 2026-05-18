import { createRouter, createWebHashHistory, type RouteLocationNormalized, type RouteRecordRaw } from 'vue-router';
import { handleHotUpdate, routes } from 'vue-router/auto-routes';
import { setupRouterProgress } from './progress';
import { routeMeta } from './routeMeta';
import './types';

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
