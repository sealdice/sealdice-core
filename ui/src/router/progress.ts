import NProgress from 'nprogress';
import type { RouteLocationNormalizedLoadedGeneric, Router } from 'vue-router';

const APP_NAME = 'SealDice';

function formatDocumentTitle(route: RouteLocationNormalizedLoadedGeneric): string {
  const pageTitle = typeof route.meta.title === 'string' ? route.meta.title.trim() : '';
  return pageTitle ? `${pageTitle} - ${APP_NAME}` : APP_NAME;
}

NProgress.configure({
  showSpinner: false,
  trickleSpeed: 120,
});

export function setupRouterProgress(router: Router): void {
  router.beforeEach(() => {
    NProgress.start();
    return true;
  });

  router.afterEach((to) => {
    document.title = formatDocumentTitle(to);
    NProgress.done();
  });

  router.onError(() => {
    NProgress.done();
  });
}
