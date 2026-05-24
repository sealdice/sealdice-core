import { ApiError } from '@/api';

// getErrorMessage 统一提取接口错误提示，避免页面层重复拆字段。
export function getErrorMessage(error: unknown, fallback = '请求失败'): string {
  if (!error) {
    return fallback;
  }

  if (error instanceof ApiError) {
    return error.message || fallback;
  }

  if (error instanceof Error) {
    return error.message || fallback;
  }

  if (typeof error === 'string' && error.trim() !== '') {
    return error;
  }

  if (typeof error === 'object') {
    const candidate = error as Record<string, unknown>;
    if (typeof candidate.message === 'string' && candidate.message.trim() !== '') {
      return candidate.message;
    }
    if (typeof candidate.detail === 'string' && candidate.detail.trim() !== '') {
      return candidate.detail;
    }
    if (typeof candidate.title === 'string' && candidate.title.trim() !== '') {
      return candidate.title;
    }
  }

  return fallback;
}
