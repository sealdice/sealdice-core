import { ApiError } from './error';

export type HttpErrorFeedback =
  | {
      kind: 'dialog';
      title: string;
      content: string;
      positiveText: string;
      negativeText?: string;
      clearSession?: boolean;
    }
  | {
      kind: 'message';
      content: string;
    };

function pickErrorMessage(error: ApiError): string {
  return error.message || error.detail || error.title || error.statusText || '请求失败';
}

export function createHttpErrorFeedback(error: ApiError): HttpErrorFeedback {
  const message = pickErrorMessage(error);

  if (error.status === 401) {
    return {
      kind: 'dialog',
      title: '身份信息',
      content: `无效的令牌。错误码 401，错误信息：${message}`,
      positiveText: '重新登录',
      negativeText: '取消',
      clearSession: true,
    };
  }

  if (error.status === 404) {
    return {
      kind: 'dialog',
      title: '接口报错',
      content:
        `检测到接口错误。错误码 404：此类错误多为接口未注册、服务未重启，` +
        `或请求路径/方法与 API 路径/方法不符。错误信息：${message}`,
      positiveText: '我知道了',
      negativeText: '取消',
    };
  }

  if (error.status === 500) {
    return {
      kind: 'dialog',
      title: '接口报错',
      content:
        `检测到接口错误。错误码 500：此类错误常见于后台 panic，请先查看后台日志。` +
        `如果影响正常使用，可清理缓存后重新登录。错误信息：${message}`,
      positiveText: '清理缓存',
      negativeText: '取消',
      clearSession: true,
    };
  }

  if (error.status === 403) {
    return {
      kind: 'message',
      content: message || '当前状态不允许执行该操作',
    };
  }

  if (error.status >= 400) {
    return {
      kind: 'message',
      content: message,
    };
  }

  return {
    kind: 'message',
    content: message,
  };
}

export function createNetworkErrorFeedback(error: unknown): HttpErrorFeedback {
  const message = error instanceof Error ? error.message : String(error || '网络错误');
  return {
    kind: 'dialog',
    title: '请求报错',
    content: `检测到请求错误：${message}`,
    positiveText: '稍后重试',
    negativeText: '取消',
  };
}
