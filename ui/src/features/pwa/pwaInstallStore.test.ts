import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';
import { usePwaInstallStore } from './pwaInstallStore';

it('原生安装提示按可用状态处理', async () => {
  const store = usePwaInstallStore(appPinia);
  store.detachListeners();

  expect(store.canInstall).toBe(false);
  expect(store.isInstalled).toBe(false);
  expect(await store.install()).toBe('unavailable');
});
