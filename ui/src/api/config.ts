function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

function normalizeBasePath(pathname: string): string {
  let normalized = pathname.trim() || '/';
  if (!normalized.startsWith('/')) {
    normalized = `/${normalized}`;
  }

  if (normalized.endsWith('/index.html')) {
    normalized = normalized.slice(0, -'/index.html'.length) || '/';
  }

  normalized = trimTrailingSlash(normalized);
  return normalized || '/';
}

export interface ApiLocationLike {
  origin: string;
  pathname: string;
}

export function resolveApiBaseUrlFromLocation(location: ApiLocationLike): string {
  const basePath = normalizeBasePath(location.pathname || '/');
  if (basePath === '/') return trimTrailingSlash(location.origin);
  return trimTrailingSlash(`${trimTrailingSlash(location.origin)}${basePath}`);
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
