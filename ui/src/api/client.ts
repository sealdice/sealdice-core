import { clearAccessToken, currentAccessToken, setAccessToken } from '@/features/auth/state';
import { queryClient } from '@/queryClient';
import { client } from './generated/client.gen';
import { getApiBaseUrl } from './config';
import { ApiError } from './error';
import { showApiFeedback } from './apiFeedback';
import { createHttpErrorFeedback, createNetworkErrorFeedback } from './httpStatusFeedback';

let configured = false;

// 配置 hey-api 生成的全局 fetch client。
// 这里是所有 HTTP 请求的共同边界：baseUrl、凭据、Bearer token、token 续期、
// 网络错误提示和 401 会话清理都在这里完成。页面和 feature 不应重复实现这些逻辑。
export function setupApiClient(): void {
  if (configured) return;
  configured = true;

  client.setConfig({
    baseUrl: getApiBaseUrl(),
    credentials: 'include',
  });

  client.interceptors.request.use((request) => {
    const token = currentAccessToken();
    if (!token) return request;

    // 后端 V2 统一使用 Authorization: Bearer。保留“已有 Authorization 不覆盖”
    // 是为了允许少数特殊请求自行控制认证头。
    const headers = new Headers(request.headers);
    if (!headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
    }

    return new Request(request, { headers });
  });

  client.interceptors.response.use((response) => {
    // 后端可通过 new-token 滚动刷新会话。刷新只更新唯一 token 源，
    // 不再维护旧版 localStorage.t。
    const newToken = response.headers.get('new-token');
    if (newToken) {
      setAccessToken(newToken);
    }

    return response;
  });

  client.interceptors.error.use((error, response, request) => {
    if (!response) {
      showApiFeedback(createNetworkErrorFeedback(error), clearApiSession);
      return error;
    }

    const apiError = new ApiError({
      status: response.status,
      statusText: response.statusText,
      data: error,
      request,
      response,
    });

    const pathname = new URL(request.url).pathname;
    // 登录接口自身的 401 只代表密码错误，不应清空当前页面其它状态；
    // 其它接口 401 才视作会话失效。
    if (response.status === 401 && pathname !== '/sd-api/v2/base/login') {
      clearApiSession();
    }

    showApiFeedback(createHttpErrorFeedback(apiError), clearApiSession);

    return apiError;
  });
}

function clearApiSession(): void {
  // QueryClient 缓存里可能有鉴权后的服务端数据，token 失效时必须一起清掉。
  clearAccessToken();
  queryClient.clear();
}
