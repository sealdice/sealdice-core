import axios, { AxiosHeaders, type AxiosRequestConfig } from 'axios';
import { useAuthStore } from '@/features/auth/store';
import { appPinia } from '@/pinia';
import { queryClient } from '@/queryClient';
import { client } from './generated/client.gen';
import { getApiBaseUrl } from './config';
import {
  ApiError,
  isRequestCanceledError,
  markApiErrorHandled,
  needsGlobalApiFeedback,
  shouldClearSessionForRequest,
} from './errors';

export { ApiError } from './errors';
export type GlobalApiFeedback = (error: ApiError) => void;
let feedback: GlobalApiFeedback | undefined;
let disposeClient: (() => void) | undefined;

// 网络层只归一化错误和管理会话；具体弹窗、主题与文案由应用入口注入。
export function setupApiClient(onGlobalError?: GlobalApiFeedback): () => void {
  if (disposeClient) return disposeClient;
  feedback = onGlobalError;
  const auth = useAuthStore(appPinia);
  const requestTokens = new WeakMap<object, string | undefined>();
  const clearSession = () => {
    auth.clearAccessToken();
    queryClient.clear();
  };
  client.setConfig({ baseURL: getApiBaseUrl(), withCredentials: true });
  const requestInterceptor = client.instance.interceptors.request.use(config => {
    const headers = AxiosHeaders.from(config.headers);
    if (auth.token && !headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${auth.token}`);
    }
    const authorization = headers.get('Authorization');
    const token =
      typeof authorization === 'string'
        ? /^Bearer\s+(.+)$/i.exec(authorization)?.[1]?.trim()
        : undefined;
    requestTokens.set(config, token);
    config.headers = headers;
    return config;
  });
  const responseInterceptor = client.instance.interceptors.response.use(
    response => {
      const newToken = response.headers['new-token'];
      const requestToken = requestTokens.get(response.config);
      // 旧请求晚到不能把新会话替换回旧令牌；首次验证存储令牌时允许续期。
      if (
        newToken &&
        (shouldClearSessionForRequest(requestToken, auth.token) ||
          (auth.isInitializing && !auth.token && requestToken))
      ) {
        auth.setAccessToken(Array.isArray(newToken) ? String(newToken[0]) : String(newToken));
      }
      return response;
    },
    error => {
      if (isRequestCanceledError(error)) return Promise.reject(error);
      const axiosError = axios.isAxiosError(error) ? error : undefined;
      const response = axiosError?.response;
      const apiError = new ApiError({
        status: response?.status ?? 0,
        statusText: response?.statusText || (error instanceof Error ? error.message : '请求失败'),
        data: response?.data ?? (error instanceof Error ? error.message : error),
        request: axiosError?.request,
        response,
      });
      const requestToken = axiosError?.config ? requestTokens.get(axiosError.config) : undefined;
      const pathname = getRequestPathname(axiosError?.config);
      if (apiError.status === 401 && pathname !== '/sd-api/v2/base/login') {
        if (!shouldClearSessionForRequest(requestToken, auth.token)) {
          markApiErrorHandled(apiError); // 过期请求既不清理新会话，也不显示误导提示。
          return Promise.reject(apiError);
        }
        clearSession();
        markApiErrorHandled(apiError); // 解锁弹窗已接管重新登录，页面无需再提示。
      }
      if (!auth.isInitializing && needsGlobalApiFeedback(apiError) && feedback) {
        feedback(apiError);
        markApiErrorHandled(apiError);
      }
      return Promise.reject(apiError);
    }
  );
  disposeClient = () => {
    client.instance.interceptors.request.eject(requestInterceptor);
    client.instance.interceptors.response.eject(responseInterceptor);
    feedback = undefined;
    disposeClient = undefined;
  };
  return disposeClient;
}

// SSE 与 HTTP 共用应用提供的网络故障展示，不引入第二套 UI 单例。
export function reportNetworkError(): void {
  feedback?.(new ApiError({ status: 0, statusText: '网络连接异常', data: null }));
}

function getRequestPathname(config?: AxiosRequestConfig): string {
  const url = config?.url ?? '';
  try {
    return new URL(url, config?.baseURL ?? getApiBaseUrl()).pathname;
  } catch {
    return url.startsWith('/') ? url : `/${url}`;
  }
}
