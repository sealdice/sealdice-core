import { computed } from 'vue';
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2ConfigAdvanced,
  getSdApiV2BaseLoginSalt,
  getSdApiV2BaseHealthQueryKey,
  postSdApiV2BaseLoginMutation,
  getSdApiV2BaseOverviewQueryKey,
  getSdApiV2BaseSecurityCheckQueryKey,
} from '@/api';
import {
  clearAccessToken,
  currentAccessToken,
  finishAuthInitialization,
  hasAccessToken,
  isInitializing,
  needsUnlock,
  setAccessToken,
} from './state';
import { passwordHash } from './crypto';

const defaultSigninPassword = 'defaultSignin';

// 认证会话 composable。它只管理 V2 token，不再兼容旧版 localStorage.t。
// 登录流程：获取盐 -> 前端 PBKDF2 哈希 -> 调 V2 login -> 写入 token -> 刷新基础查询。
export function useAuthSession() {
  const queryClient = useQueryClient();
  const signinMutation = useMutation(postSdApiV2BaseLoginMutation());

  const clearSession = () => {
    // 登出或鉴权失效时必须清空 Vue Query，防止旧账号/旧 token 的缓存继续显示。
    clearAccessToken();
    queryClient.clear();
  };

  const signinWithHash = async (password: string) => {
    const result = await signinMutation.mutateAsync({
      body: {
        password,
      },
    });
    const token = result.item.token;
    setAccessToken(token);
    await queryClient.invalidateQueries({ queryKey: getSdApiV2BaseHealthQueryKey() });
    await queryClient.invalidateQueries({ queryKey: getSdApiV2BaseOverviewQueryKey() });
    await queryClient.invalidateQueries({ queryKey: getSdApiV2BaseSecurityCheckQueryKey() });
    return token;
  };

  const signin = async (input: { password: string }) => {
    const salt = await getSdApiV2BaseLoginSalt({ throwOnError: true });
    const hashedPassword = await passwordHash(salt.data.item.salt, input.password);
    return signinWithHash(hashedPassword);
  };

  const tryDefaultSignin = async () => {
    // 首次启动未设置密码时，后端允许 defaultSignin 快速进入后台。
    // 已有 token 也要先用受保护接口验证，避免把旧核心/旧会话的 token 当成有效会话。
    try {
      const storedToken = currentAccessToken();
      if (storedToken) {
        // 校验期间先撤销全局会话，阻止页面查询和实时连接继续使用待验证的旧 token。
        clearSession();
        try {
          await getSdApiV2ConfigAdvanced({
            headers: { Authorization: `Bearer ${storedToken}` },
            throwOnError: true,
          });
          // 保留响应拦截器可能返回的新 token；没有刷新时恢复已验证的 token。
          setAccessToken(currentAccessToken() || storedToken);
          return true;
        } catch {
          // token 无效或服务暂不可用时，继续尝试无密码默认登录。
          clearSession();
        }
      }
      await signinWithHash(defaultSigninPassword);
      return true;
    } catch {
      clearSession();
      return false;
    } finally {
      finishAuthInitialization();
    }
  };

  return {
    hasAccessToken,
    isInitializing,
    needsUnlock,
    currentUser: computed(() => null),
    currentUserErrorText: computed(() => ''),
    signinMutation,
    signoutMutation: signinMutation,
    signin,
    signinWithHash,
    tryDefaultSignin,
    signout: clearSession,
    clearSession,
  };
}
