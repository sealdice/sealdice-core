import { computed } from 'vue';
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2BaseLoginSalt,
  getSdApiV2BaseHealthQueryKey,
  postSdApiV2BaseLoginMutation,
  getSdApiV2BaseOverviewQueryKey,
  getSdApiV2BaseSecurityCheckQueryKey,
} from '@/api';
import { clearAccessToken, currentAccessToken, hasAccessToken, setAccessToken } from './state';
import { passwordHash } from './crypto';

const defaultSigninPassword = 'defaultSignin';

export function useAuthSession() {
  const queryClient = useQueryClient();
  const signinMutation = useMutation(postSdApiV2BaseLoginMutation());

  const clearSession = () => {
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
    if (currentAccessToken()) {
      return true;
    }
    try {
      await signinWithHash(defaultSigninPassword);
      return true;
    } catch {
      return false;
    }
  };

  return {
    hasAccessToken,
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
