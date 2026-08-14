import axios, { AxiosHeaders, type AxiosError, type AxiosRequestConfig } from 'axios';
import { createDiscreteApi } from 'naive-ui';
import { h, type VNodeChild } from 'vue';
import {
  clearAccessToken,
  currentAccessToken,
  isInitializing,
  setAccessToken,
} from '@/features/auth/state';
import { queryClient } from '@/queryClient';
import { TEST_MODE_ERROR_CODE, TEST_MODE_DEFAULT_REASON } from '@/features/testMode/state';
import { client } from './generated/client.gen';
import { getApiBaseUrl } from './config';

let configured = false;
const { dialog, message } = createDiscreteApi(['dialog', 'message']);
const requestTokens = new WeakMap<object, string | undefined>();

type ApiErrorInit = {
  status: number;
  statusText: string;
  data: unknown;
  request?: unknown;
  response?: unknown;
};

export type ApiErrorFeedback =
  | {
      kind: 'dialog';
      title: string;
      content: () => VNodeChild;
      positiveText: string;
      negativeText?: string;
      clearSession?: boolean;
    }
  | {
      kind: 'message';
      content: string;
    }
  | {
      kind: 'business';
      content: string;
    };

let activeDialogKey = '';

function pickMessage(data: unknown, fallback: string): string {
  if (!data) return fallback;

  if (typeof data === 'string') return data;

  if (typeof data === 'object') {
    const candidate = data as Record<string, unknown>;
    if (typeof candidate.message === 'string' && candidate.message.trim() !== '') {
      return candidate.message;
    }
    if (typeof candidate.detail === 'string' && candidate.detail.trim() !== '') {
      return candidate.detail;
    }
    if (typeof candidate.error === 'string' && candidate.error.trim() !== '') {
      return candidate.error;
    }
  }

  return fallback;
}

export class ApiError extends Error {
  code?: string;
  detail?: string;
  title?: string;
  status: number;
  statusText: string;
  data: unknown;
  request?: unknown;
  response?: unknown;

  constructor(init: ApiErrorInit) {
    const message = pickMessage(init.data, init.statusText);
    super(message);
    this.name = 'ApiError';

    if (typeof init.data === 'object' && init.data) {
      const candidate = init.data as Record<string, unknown>;
      if (typeof candidate.code === 'string') this.code = candidate.code;
      if (typeof candidate.detail === 'string') this.detail = candidate.detail;
      if (typeof candidate.title === 'string') this.title = candidate.title;
    }

    this.status = init.status;
    this.statusText = init.statusText;
    this.data = init.data;
    this.request = init.request;
    this.response = init.response;
  }
}

export function isRequestCanceledError(error: unknown): boolean {
  if (axios.isCancel(error)) return true;
  if (!error || typeof error !== 'object') return false;

  const candidate = error as { code?: unknown; name?: unknown };
  return candidate.code === 'ERR_CANCELED' || candidate.name === 'CanceledError';
}

export function shouldClearSessionForRequest(
  requestToken: string | undefined,
  currentToken: string
): boolean {
  return requestToken !== undefined && requestToken === currentToken;
}

// 配置 hey-api 生成的全局 axios client。
// 这里是所有 HTTP 请求的共同边界：baseURL、凭据、Bearer token、token 续期、
// 网络/会话级错误提示和 401 会话清理都在这里完成。业务级 4xx 交给页面处理。
export function setupApiClient(): void {
  if (configured) return;
  configured = true;

  client.setConfig({
    baseURL: getApiBaseUrl(),
    withCredentials: true,
  });

  client.instance.interceptors.request.use(config => {
    const token = currentAccessToken();
    // 后端 V2 统一使用 Authorization: Bearer。保留“已有 Authorization 不覆盖”
    // 是为了允许少数特殊请求自行控制认证头。
    const headers = AxiosHeaders.from(config.headers);
    const requestToken = headers.has('Authorization')
      ? readBearerToken(headers)
      : token || undefined;
    requestTokens.set(config, requestToken);

    if (!token) return config;

    if (!headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${token}`);
      config.headers = headers;
    }

    return config;
  });

  client.instance.interceptors.response.use(
    response => {
      // 后端可通过 new-token 滚动刷新会话。刷新只更新唯一 token 源，
      // 不再维护旧版 localStorage.t。
      const newToken = response.headers['new-token'];
      if (newToken) {
        setAccessToken(Array.isArray(newToken) ? String(newToken[0]) : String(newToken));
      }

      return response;
    },
    error => {
      // 路由切换会让 TanStack Query 取消仍在进行中的请求，这属于正常控制流，
      // 不应再被用户看到为 “Canceled” 弹窗或错误提示。
      if (isRequestCanceledError(error)) {
        return Promise.reject(error);
      }

      const axiosError = axios.isAxiosError(error) ? error : undefined;
      const requestToken = axiosError?.config ? requestTokens.get(axiosError.config) : undefined;
      const clearSessionForRequest = () => {
        if (shouldClearSessionForRequest(requestToken, currentAccessToken())) {
          clearApiSession();
        }
      };

      if (!axiosError?.response) {
        if (!isInitializing.value) {
          showApiFeedback(createNetworkErrorFeedback(error), clearSessionForRequest);
        }
        return Promise.reject(error);
      }

      const apiError = toApiError(axiosError);
      const pathname = getRequestPathname(axiosError.config);

      // 登录接口自身的 401 只代表密码错误，不应清空当前页面其它状态；
      // 其它接口 401 才视作会话失效。
      const isUnauthorized = apiError.status === 401 && pathname !== '/sd-api/v2/base/login';
      const isCurrentSessionRequest = shouldClearSessionForRequest(
        requestToken,
        currentAccessToken()
      );
      if (isUnauthorized) {
        // 旧请求晚到时不能清理已经建立的新会话，也不应再弹出误导性的 401 对话框。
        if (!isCurrentSessionRequest) {
          return Promise.reject(apiError);
        }
        clearApiSession();
      }
      if (!isInitializing.value) {
        showApiFeedback(createApiErrorFeedback(apiError, { pathname }), clearSessionForRequest);
      }

      return Promise.reject(apiError);
    }
  );
}

function clearApiSession(): void {
  // QueryClient 缓存里可能有鉴权后的服务端数据，token 失效时必须一起清掉。
  clearAccessToken();
  queryClient.clear();
}

function pickErrorMessage(error: ApiError): string {
  return error.message || error.detail || error.title || error.statusText || '请求失败';
}

function readBearerToken(headers: AxiosHeaders): string | undefined {
  const authorization = headers.get('Authorization');
  if (typeof authorization !== 'string') return undefined;

  const match = /^Bearer\s+(.+)$/i.exec(authorization);
  return match?.[1]?.trim() ?? '';
}

function isTestModeBlockedError(error: ApiError): boolean {
  const data =
    typeof error.data === 'object' && error.data ? (error.data as Record<string, unknown>) : null;
  return (
    error.code === TEST_MODE_ERROR_CODE ||
    data?.testMode === true ||
    error.message === TEST_MODE_DEFAULT_REASON ||
    error.detail === TEST_MODE_DEFAULT_REASON
  );
}

function formatError(error: unknown): string {
  return String(error || '请求失败');
}

function errorCodeNode(code: number): VNodeChild {
  return h('span', { style: { color: 'red' } }, ` ${code} `);
}

function dialogParagraph(parts: Array<string | number>, key: number): VNodeChild {
  return h(
    'p',
    { key },
    parts.map((part, index) => {
      if (typeof part === 'number') return errorCodeNode(part);
      return h('span', { key: index }, part);
    })
  );
}

function dialogContent(paragraphs: Array<Array<string | number>>): () => VNodeChild {
  return () =>
    h(
      'div',
      { class: 'api-error-dialog-content' },
      paragraphs.map((paragraph, index) => dialogParagraph(paragraph, index))
    );
}

export function createApiErrorFeedback(
  error: ApiError,
  options: { pathname?: string } = {}
): ApiErrorFeedback {
  const errorMessage = pickErrorMessage(error);

  if (isTestModeBlockedError(error)) {
    return {
      kind: 'business',
      content: errorMessage || TEST_MODE_DEFAULT_REASON,
    };
  }

  if (error.status === 401) {
    if (options.pathname === '/sd-api/v2/base/login') {
      return {
        kind: 'business',
        content: errorMessage,
      };
    }

    return {
      kind: 'dialog',
      title: '身份信息',
      content: dialogContent([
        ['无效的令牌'],
        ['错误码：', 401, `错误信息：${formatError(error)}`],
      ]),
      positiveText: '重新登录',
      negativeText: '取消',
      clearSession: true,
    };
  }

  if (error.status === 404) {
    return {
      kind: 'dialog',
      title: '接口报错',
      content: dialogContent([
        [`检测到接口错误${formatError(error)}`],
        [
          '错误码',
          404,
          '：此类错误多为接口未注册（或未重启）或者请求路径（方法）与api路径（方法）不符--如果为自动化代码请检查是否存在空格',
        ],
      ]),
      positiveText: '我知道了',
      negativeText: '取消',
    };
  }

  if (error.status === 500) {
    return {
      kind: 'dialog',
      title: '接口报错',
      content: dialogContent([
        [`检测到接口错误${formatError(error)}`],
        [
          '错误码',
          500,
          '：此类错误内容常见于后台panic，请先查看后台日志，如果影响您正常使用可强制登出清理缓存',
        ],
      ]),
      positiveText: '清理缓存',
      negativeText: '取消',
      clearSession: true,
    };
  }

  if (error.status >= 400) {
    return {
      kind: 'business',
      content: errorMessage,
    };
  }

  return {
    kind: 'message',
    content: errorMessage,
  };
}

export function createNetworkErrorFeedback(_error?: unknown): ApiErrorFeedback {
  void _error;
  return {
    kind: 'dialog',
    title: '请求报错',
    content: dialogContent([['正在自动重连，页面数据可能暂时不是最新，请检查后台是否正常运行。']]),
    positiveText: '稍后重试',
    negativeText: '取消',
  };
}

export function showNetworkErrorDialog(): void {
  showApiFeedback(createNetworkErrorFeedback(), () => undefined);
}

function showDialog(
  feedback: Extract<ApiErrorFeedback, { kind: 'dialog' }>,
  onClearSession: () => void
) {
  const key = `${feedback.title}:${pickDialogText(feedback.content())}`;
  if (activeDialogKey === key) return;

  activeDialogKey = key;
  dialog.warning({
    title: feedback.title,
    content: feedback.content,
    positiveText: feedback.positiveText,
    negativeText: feedback.negativeText,
    maskClosable: false,
    onPositiveClick: () => {
      if (feedback.clearSession) {
        onClearSession();
      }
    },
    onAfterLeave: () => {
      if (activeDialogKey === key) {
        activeDialogKey = '';
      }
    },
  });
}

function showApiFeedback(feedback: ApiErrorFeedback, onClearSession: () => void): void {
  if (feedback.kind === 'dialog') {
    showDialog(feedback, onClearSession);
    return;
  }

  if (feedback.kind === 'business') {
    return;
  }

  message.error(feedback.content, {
    closable: true,
    duration: 5000,
  });
}

function pickDialogText(content: VNodeChild): string {
  if (typeof content === 'string' || typeof content === 'number') return String(content);
  if (Array.isArray(content)) return content.map(pickDialogText).join('');
  if (content && typeof content === 'object' && 'children' in content) {
    return pickDialogText(content.children as VNodeChild);
  }
  return '';
}

function toApiError(error: AxiosError): ApiError {
  const response = error.response;
  return new ApiError({
    status: response?.status ?? 0,
    statusText: response?.statusText || error.message || '请求失败',
    data: response?.data ?? error.message,
    request: error.request,
    response,
  });
}

function getRequestPathname(config?: AxiosRequestConfig): string {
  const url = config?.url ?? '';
  const baseURL = config?.baseURL ?? getApiBaseUrl();
  try {
    return new URL(url, baseURL).pathname;
  } catch {
    return url.startsWith('/') ? url : `/${url}`;
  }
}
