function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

export interface ApiLocationLike {
  origin: string;
  pathname: string;
}

export function resolveApiBaseUrlFromLocation(location: ApiLocationLike): string {
  return trimTrailingSlash(location.origin);
}

export function joinApiBasePath(baseUrl: string, path: string): string {
  const normalizedBase = trimTrailingSlash(baseUrl);
  const normalizedPath = path.replace(/^\/+/, '');
  if (!normalizedBase) return `/${normalizedPath}`;
  return `${normalizedBase}/${normalizedPath}`;
}

// 切换到旧版 UI：走 /?ui=old，服务端会记住偏好并跳到 /old-ui/。
export function resolveSwitchToOldUIUrl(location: ApiLocationLike): string {
  return `${trimTrailingSlash(location.origin)}/?ui=old`;
}

/**
 * API Base URL.
 *
 * 前端统一走同源地址，开发态由 Vite 代理映射到后端，
 * 生产态则复用当前站点 origin。
 */
export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined' && typeof window.location?.origin === 'string') {
    return resolveApiBaseUrlFromLocation(window.location);
  }

  return '';
}
