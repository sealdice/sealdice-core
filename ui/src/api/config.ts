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

// 切换到旧版 UI：保留反向代理部署的路径前缀，服务端会记住偏好并跳到 /old-ui/。
export function resolveSwitchToOldUIUrl(location: ApiLocationLike): string {
  const v2UIPathIndex = location.pathname.lastIndexOf('/v2ui');
  const deploymentPrefix =
    v2UIPathIndex >= 0 &&
    (location.pathname.length === v2UIPathIndex + '/v2ui'.length ||
      location.pathname[v2UIPathIndex + '/v2ui'.length] === '/')
      ? location.pathname.slice(0, v2UIPathIndex)
      : '';
  return `${trimTrailingSlash(location.origin)}${deploymentPrefix || ''}/?ui=old`;
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
