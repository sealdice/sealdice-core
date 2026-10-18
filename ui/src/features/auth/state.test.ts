import { expect, it } from 'vitest';
import { createPinia, disposePinia } from 'pinia';
import { useAuthStore } from './store';

it('初始化结束前不弹解锁框，登录和退出同步令牌状态', () => {
  const pinia = createPinia();
  const auth = useAuthStore(pinia);
  auth.clearAccessToken();
  expect(auth.isInitializing).toBe(true);
  expect(auth.needsUnlock).toBe(false);
  auth.finishInitialization();
  expect(auth.needsUnlock).toBe(true);
  auth.setAccessToken(' token-1 ');
  expect(auth.currentAccessToken()).toBe('token-1');
  expect(auth.hasAccessToken).toBe(true);
  expect(auth.needsUnlock).toBe(false);
  auth.clearAccessToken();
  expect(auth.token).toBe('');
  expect(auth.hasAccessToken).toBe(false);
  disposePinia(pinia);
});
