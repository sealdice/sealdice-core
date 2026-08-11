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

export function resolveOldUIUrlFromLocation(location: ApiLocationLike): string {
  return joinApiBasePath(resolveApiBaseUrlFromLocation(location), '/old-ui/');
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
