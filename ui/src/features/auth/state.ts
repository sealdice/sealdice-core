import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useAuthStore } from './store';

const authStore = useAuthStore(appPinia);
const { hasAccessToken } = storeToRefs(authStore);

// 兼容层：旧代码仍从 state.ts 读取认证状态，新代码优先直接使用 useAuthStore。
export function currentAccessToken(): string {
  return authStore.currentAccessToken();
}

// hasAccessToken 用于统一判断当前是否存在登录态。
export { hasAccessToken };

// setAccessToken 同步更新内存和本地存储中的 access token。
export function setAccessToken(token: string): void {
  authStore.setAccessToken(token);
}

// clearAccessToken 清理当前 access token。
export function clearAccessToken(): void {
  authStore.clearAccessToken();
}
