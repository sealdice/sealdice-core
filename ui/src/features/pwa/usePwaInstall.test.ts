import { it } from 'vitest';
import { appPinia } from '@/pinia';
import { usePwaInstallStore } from './pwaInstallStore';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) {
      throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
    }
  };

  const store = usePwaInstallStore(appPinia);
  store.detachListeners();

  assertEqual(store.canInstall, false);
  assertEqual(store.isInstalled, false);
  assertEqual(await store.install(), 'unavailable');
});
