function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

/**
 * API Base URL.
 *
 * 前端统一走同源地址，开发态由 Vite 代理映射到后端，
 * 生产态则复用当前站点 origin。
 */
export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined' && typeof window.location?.origin === 'string') {
    return trimTrailingSlash(window.location.origin);
  }

  return '';
}
