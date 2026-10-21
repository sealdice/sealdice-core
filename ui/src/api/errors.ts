import axios from 'axios';

export const TEST_MODE_ERROR_CODE = 'TEST_MODE_BLOCKED';

export class ApiError extends Error {
  code?: string;
  detail?: string;
  title?: string;
  status: number;
  statusText: string;
  data: unknown;
  request?: unknown;
  response?: unknown;

  constructor(init: {
    status: number;
    statusText: string;
    data: unknown;
    request?: unknown;
    response?: unknown;
  }) {
    const data =
      init.data && typeof init.data === 'object' ? (init.data as Record<string, unknown>) : {};
    const message = [data.message, data.detail, data.error, init.data, init.statusText].find(
      value => typeof value === 'string' && value.trim()
    );
    super(typeof message === 'string' ? message : '请求失败');
    this.name = 'ApiError';
    Object.assign(this, init);
    this.status = init.status;
    this.statusText = init.statusText;
    this.data = init.data;
    if (typeof data.code === 'string') this.code = data.code;
    if (typeof data.detail === 'string') this.detail = data.detail;
    if (typeof data.title === 'string') this.title = data.title;
  }
}

// 只标记已经交给全局展示的错误；局部校验和业务失败仍由调用方反馈。
const handledErrors = new WeakSet<object>();
export function markApiErrorHandled(error: object): void {
  handledErrors.add(error);
}
export function isApiErrorHandled(error: unknown): boolean {
  return (
    isRequestCanceledError(error) ||
    (typeof error === 'object' && error !== null && handledErrors.has(error))
  );
}

export function isRequestCanceledError(error: unknown): boolean {
  if (axios.isCancel(error)) return true;
  if (!error || typeof error !== 'object') return false;
  const value = error as { code?: unknown; name?: unknown };
  return (
    value.code === 'ERR_CANCELED' || value.name === 'CanceledError' || value.name === 'AbortError'
  );
}

export function shouldClearSessionForRequest(
  token: string | undefined,
  currentToken: string
): boolean {
  return token !== undefined && token === currentToken;
}

export function needsGlobalApiFeedback(error: ApiError): boolean {
  const data =
    error.data && typeof error.data === 'object' ? (error.data as Record<string, unknown>) : {};
  if (error.code === TEST_MODE_ERROR_CODE || data.testMode === true) return false;
  // 401 会清理会话并触发 AppUnlockDialog，由解锁弹窗统一引导重新登录，
  // 不再额外弹出一次“身份信息/Unauthorized”对话框。
  return [0, 404, 500].includes(error.status);
}
