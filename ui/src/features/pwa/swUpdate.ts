import { registerSW } from 'virtual:pwa-register';
import { appPinia } from '@/pinia';
import {
  CONTROLLED_RELOAD_AT_KEY,
  parseReloadGuardTimestamp,
  shouldAllowControlledReload,
} from './swUpdateState';
import { useSwUpdateStore } from './swUpdateStore';

// 长期开着的控制台页面不会主动发现新版本，注册成功后按周期 + 可见性变化检查。
const SW_UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000;

// 受控刷新需要跳过 beforeunload 的未保存拦截：
// 用户已经在升级提示里确认过“丢弃更改并刷新”，不应再弹一次浏览器原生确认框。
let unloadBypassArmed = false;

export function isControlledReloadArmed(): boolean {
  return unloadBypassArmed;
}

// 刷新前用 replaceState 把 URL 指到目标路由（不触发 hashchange，旧路由不会二次导航），
// 刷新后 hash history 直接落在用户本来想去的页面。
// 返回 false 表示处于防循环窗口期内，调用方应改为提示用户手动刷新。
export function requestControlledReload(targetPath?: string | null): boolean {
  if (typeof window === 'undefined') return false;

  const now = Date.now();
  const lastReloadAt = parseReloadGuardTimestamp(
    window.sessionStorage.getItem(CONTROLLED_RELOAD_AT_KEY)
  );
  if (!shouldAllowControlledReload(lastReloadAt, now)) return false;

  window.sessionStorage.setItem(CONTROLLED_RELOAD_AT_KEY, String(now));

  if (targetPath) {
    try {
      window.history.replaceState(null, '', `#${targetPath}`);
    } catch {
      // replaceState 失败时退化为原地刷新，不影响恢复主流程。
    }
  }

  unloadBypassArmed = true;
  window.location.reload();
  return true;
}

export function setupPwaUpdateHandling(): void {
  if (typeof window === 'undefined') return;

  const swUpdateStore = useSwUpdateStore(appPinia);

  // autoUpdate 模式下新 SW 会 skipWaiting + clientsClaim 并清掉旧预缓存，
  // 旧页面继续运行必然与新缓存错配，收到 onNeedReload 必须刷新对齐版本。
  // 不传 onNeedReload 时插件默认直接 window.location.reload()，绕过未保存变更检查。
  registerSW({
    immediate: true,
    onNeedReload() {
      swUpdateStore.markUpdateReady('sw-update');
    },
    onRegisteredSW(_swUrl, registration) {
      if (!registration) return;
      const checkForUpdate = () => {
        void registration.update().catch(() => {});
      };
      window.setInterval(checkForUpdate, SW_UPDATE_CHECK_INTERVAL_MS);
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') checkForUpdate();
      });
    },
    onRegisterError(error) {
      console.error('[pwa] service worker 注册失败', error);
    },
  });
}
