import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import { useAuthStore } from './store';

const authStore = useAuthStore(appPinia);
const { hasAccessToken, isInitializing, needsUnlock } = storeToRefs(authStore);

// 兼容层：旧代码仍从 state.ts 读取认证状态，新代码优先直接使用 useAuthStore。
export function currentAccessToken(): string {
  return authStore.currentAccessToken();
}

// 认证初始化完成前，UI 应等待默认登录结果再决定是否要求输入密码。
export { hasAccessToken, isInitializing, needsUnlock };

// setAccessToken 同步更新内存和本地存储中的 access token。
export function setAccessToken(token: string): void {
  authStore.setAccessToken(token);
}

// clearAccessToken 清理当前 access token。
export function clearAccessToken(): void {
  authStore.clearAccessToken();
}

export function finishAuthInitialization(): void {
  authStore.finishInitialization();
}
