function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

function findV2UIPathSegment(pathname: string): number {
  const marker = '/v2ui';
  let index = pathname.indexOf(marker);

  while (index >= 0) {
    const nextChar = pathname[index + marker.length];
    if (nextChar === undefined || nextChar === '/') {
      return index;
    }
    index = pathname.indexOf(marker, index + marker.length);
  }

  return -1;
}

export interface ApiLocationLike {
  origin: string;
  pathname: string;
}

export function resolveApiBaseUrlFromLocation(location: ApiLocationLike): string {
  const pathname = location.pathname || '/';
  const v2UIIndex = findV2UIPathSegment(pathname);
  if (v2UIIndex < 0) return trimTrailingSlash(location.origin);
  return trimTrailingSlash(`${trimTrailingSlash(location.origin)}${pathname.slice(0, v2UIIndex)}`);
}

export function joinApiBasePath(baseUrl: string, path: string): string {
  const normalizedBase = trimTrailingSlash(baseUrl);
  const normalizedPath = path.replace(/^\/+/, '');
  if (!normalizedBase) return `/${normalizedPath}`;
  return `${normalizedBase}/${normalizedPath}`;
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
