import { computed, readonly, ref } from 'vue';
import { clearLegacyAccessToken, getLegacyAccessToken, setLegacyAccessToken } from '@/api/legacy';
import { queryClient } from '@/queryClient';
import { getHello } from '@/features/runtime/legacyApi';
import { passwordHash } from './crypto';
import { getLegacySalt, legacySignin } from './legacyApi';

const defaultSigninPassword = 'defaultSignin';

const legacyAccessTokenRef = ref(getLegacyAccessToken());
const saltRef = ref('');
const isPendingRef = ref(false);
const lastErrorRef = ref<unknown>(null);

function syncLegacyAccessToken(): string {
  legacyAccessTokenRef.value = getLegacyAccessToken();
  return legacyAccessTokenRef.value;
}

async function loadSalt(): Promise<string> {
  const result = await getLegacySalt();
  saltRef.value = result.salt;
  return result.salt;
}

function clearLegacySession(): void {
  clearLegacyAccessToken();
  legacyAccessTokenRef.value = '';
  queryClient.clear();
}

async function signinWithHash(password: string): Promise<string> {
  isPendingRef.value = true;
  lastErrorRef.value = null;
  try {
    const result = await legacySignin(password);
    setLegacyAccessToken(result.token);
    legacyAccessTokenRef.value = result.token;
    return result.token;
  } catch (error) {
    lastErrorRef.value = error;
    throw error;
  } finally {
    isPendingRef.value = false;
  }
}

async function signinWithPassword(password: string): Promise<string> {
  const salt = saltRef.value || (await loadSalt());
  const hashedPassword = await passwordHash(salt, password);
  return signinWithHash(hashedPassword);
}

async function tryAutoSignin(): Promise<boolean> {
  isPendingRef.value = true;
  lastErrorRef.value = null;
  try {
    await loadSalt();
    const token = syncLegacyAccessToken();
    if (token) {
      try {
        await getHello();
        return true;
      } catch {
        // 保持旧项目行为：已有 token 校验失败后继续尝试 defaultSignin，但不提前清掉本地 token。
      }
    }

    await signinWithHash(defaultSigninPassword);
    return legacyAccessTokenRef.value !== '';
  } catch (error) {
    lastErrorRef.value = error;
    return false;
  } finally {
    isPendingRef.value = false;
  }
}

export function useLegacyAuthSession() {
  return {
    legacyAccessToken: readonly(legacyAccessTokenRef),
    hasLegacyAccessToken: computed(() => legacyAccessTokenRef.value !== ''),
    salt: readonly(saltRef),
    isPending: readonly(isPendingRef),
    lastError: readonly(lastErrorRef),
    loadSalt,
    signinWithPassword,
    signinWithHash,
    tryAutoSignin,
    clearLegacySession,
    syncLegacyAccessToken,
  };
}
