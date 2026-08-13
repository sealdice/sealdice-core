import { it } from 'vitest';
import {
  clearAccessToken,
  currentAccessToken,
  finishAuthInitialization,
  hasAccessToken,
  isInitializing,
  needsUnlock,
  setAccessToken,
} from './state';
import { useAuthStore } from './store';
import { appPinia } from '@/pinia';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) {
      throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
    }
  };

  const store = useAuthStore(appPinia);
  store.clearAccessToken();

  assertEqual(hasAccessToken.value, false);
  assertEqual(currentAccessToken(), '');
  assertEqual(isInitializing.value, true);
  assertEqual(needsUnlock.value, false);

  finishAuthInitialization();
  assertEqual(isInitializing.value, false);
  assertEqual(needsUnlock.value, true);

  setAccessToken(' token-1 ');
  assertEqual(store.token, 'token-1');
  assertEqual(hasAccessToken.value, true);
  assertEqual(currentAccessToken(), 'token-1');
  assertEqual(needsUnlock.value, false);

  clearAccessToken();
  assertEqual(store.token, '');
  assertEqual(hasAccessToken.value, false);
});
