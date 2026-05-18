import { computed, ref } from 'vue';

const accessTokenStorageKey = 'token';
const accessTokenRef = ref(readTokenFromStorage());

function readTokenFromStorage(): string {
  try {
    return localStorage.getItem(accessTokenStorageKey)?.trim() ?? '';
  } catch {
    return '';
  }
}

// currentAccessToken 返回当前内存中的 access token。
export function currentAccessToken(): string {
  return accessTokenRef.value;
}

// hasAccessToken 用于统一判断当前是否存在登录态。
export const hasAccessToken = computed(() => accessTokenRef.value !== '');

// setAccessToken 同步更新内存和本地存储中的 access token。
export function setAccessToken(token: string): void {
  const normalized = token.trim();
  accessTokenRef.value = normalized;

  try {
    if (normalized) {
      localStorage.setItem(accessTokenStorageKey, normalized);
    } else {
      localStorage.removeItem(accessTokenStorageKey);
    }
  } catch {
    // 忽略本地存储异常，避免阻断主流程。
  }
}

// clearAccessToken 清理当前 access token。
export function clearAccessToken(): void {
  setAccessToken('');
}
