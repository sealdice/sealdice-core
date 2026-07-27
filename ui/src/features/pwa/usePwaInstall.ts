import { storeToRefs } from 'pinia';
import { onBeforeUnmount, onMounted } from 'vue';
import { appPinia } from '@/pinia';
import { usePwaInstallStore } from './pwaInstallStore';

const pwaInstallStore = usePwaInstallStore(appPinia);
const pwaInstallRefs = storeToRefs(pwaInstallStore);

export function usePwaInstall() {
  onMounted(() => {
    pwaInstallStore.retainListeners();
  });
  onBeforeUnmount(() => {
    pwaInstallStore.releaseListeners();
  });

  return {
    // 兼容层：组件仍通过 usePwaInstall() 使用，PWA 安装状态集中到 Pinia。
    isSupported: pwaInstallRefs.isSupported,
    canInstall: pwaInstallRefs.canInstall,
    isInstalled: pwaInstallRefs.isInstalled,
    installing: pwaInstallRefs.installing,
    install: pwaInstallStore.install,
  };
}
