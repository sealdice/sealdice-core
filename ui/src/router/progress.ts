import NProgress from 'nprogress';
import type { Router } from 'vue-router';

NProgress.configure({
  showSpinner: false,
  trickleSpeed: 120,
});

export function setupRouterProgress(router: Router): void {
  router.beforeEach(() => {
    NProgress.start();
    return true;
  });

  router.afterEach(() => {
    NProgress.done();
  });

  router.onError(() => {
    NProgress.done();
  });
}
