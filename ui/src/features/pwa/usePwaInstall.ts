import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { usePwaInstallStore } from './pwaInstallStore';

const pwaInstallStore = usePwaInstallStore(appPinia);
const pwaInstallRefs = storeToRefs(pwaInstallStore);

export function setupPwaInstallHandling(): void {
  // 安装事件可能早于侧边栏挂载；应用启动时监听，避免丢失 Chromium 的一次性 prompt。
  pwaInstallStore.attachListeners();
}

export function usePwaInstall() {
  return {
    canInstall: pwaInstallRefs.canInstall,
    isInstalled: pwaInstallRefs.isInstalled,
    installing: pwaInstallRefs.installing,
    install: pwaInstallStore.install,
  };
}
