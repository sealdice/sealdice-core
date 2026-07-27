import { it } from 'vitest';
import { hasAccessToken, clearAccessToken, currentAccessToken, setAccessToken } from './state';
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

  setAccessToken(' token-1 ');
  assertEqual(store.token, 'token-1');
  assertEqual(hasAccessToken.value, true);
  assertEqual(currentAccessToken(), 'token-1');

  clearAccessToken();
  assertEqual(store.token, '');
  assertEqual(hasAccessToken.value, false);
});
