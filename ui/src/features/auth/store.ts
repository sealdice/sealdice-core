import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';

const accessTokenStorageKey = 'token';
const storage = typeof window === 'undefined' ? undefined : window.localStorage;

function readTokenFromStorage(): string {
  try {
    return storage?.getItem(accessTokenStorageKey)?.trim() ?? '';
  } catch {
    return '';
  }
}

function writeTokenToStorage(token: string): void {
  try {
    if (token) {
      storage?.setItem(accessTokenStorageKey, token);
    } else {
      storage?.removeItem(accessTokenStorageKey);
    }
  } catch {
    // 本地存储失败时只影响持久化，不阻断当前会话。
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = shallowRef(readTokenFromStorage());
  const hasAccessToken = computed(() => token.value !== '');

  function currentAccessToken(): string {
    return token.value;
  }

  function setAccessToken(nextToken: string): void {
    const normalized = nextToken.trim();
    token.value = normalized;
    writeTokenToStorage(normalized);
  }

  function clearAccessToken(): void {
    setAccessToken('');
  }

  return {
    token,
    hasAccessToken,
    currentAccessToken,
    setAccessToken,
    clearAccessToken,
  };
});
