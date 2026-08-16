import type { Router } from 'vue-router';
import { appPinia } from '@/pinia';
import { isChunkLoadError } from '@/features/pwa/swUpdateState';
import { useSwUpdateStore } from '@/features/pwa/swUpdateStore';

// 页面 chunk 是带 hash 的异步 import，版本切换后旧文件会 404。
// 把这类失败统一上报到升级提示，避免“点击侧边栏没反应、停留在当前页”。
export function setupChunkLoadRecovery(router: Router): void {
  const swUpdateStore = useSwUpdateStore(appPinia);

  // vite:preloadError 不带路由上下文，记录最近一次导航目标用于恢复后跳转。
  let pendingTarget: string | null = null;

  router.beforeEach(to => {
    pendingTarget = to.fullPath;
    return true;
  });

  router.afterEach(() => {
    pendingTarget = null;
  });

  router.onError((error, to) => {
    if (!isChunkLoadError(error)) return;
    swUpdateStore.markUpdateReady('chunk-error', to.fullPath);
  });

  if (typeof window === 'undefined') return;

  // 路由之外的动态 import（如 defineAsyncComponent）失败只会触发 vite:preloadError。
  window.addEventListener('vite:preloadError', event => {
    // 阻止 Vite 再次抛出，恢复流程统一走升级提示。
    event.preventDefault();
    swUpdateStore.markUpdateReady(
      'chunk-error',
      pendingTarget ?? router.currentRoute.value.fullPath
    );
  });
}
