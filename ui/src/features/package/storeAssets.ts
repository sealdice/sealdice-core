import type { StorePackage } from '@/api';

function splitPackageId(id: string): { namespace: string; name: string } | null {
  const separator = id.indexOf('/');
  if (separator <= 0 || separator === id.length - 1) return null;
  return { namespace: id.slice(0, separator), name: id.slice(separator + 1) };
}

function isAbsoluteAssetUrl(value: string): boolean {
  return /^(https?:)?\/\//i.test(value) || /^(data|blob):/i.test(value);
}

/**
 * 商店图标可能声明为绝对 URL，也可能是包内相对路径。
 * 相对路径交给后端代理读取，避免前端直接请求商店后端时的跨域和鉴权问题。
 */
export function getStorePackageIconUrl(pkg: StorePackage): string {
  const icon = pkg.storeAssets?.icon?.trim();
  if (!icon) return '';
  if (isAbsoluteAssetUrl(icon)) return icon;

  if (icon.startsWith('/')) {
    try {
      return new URL(icon, pkg.download?.url ?? '').toString();
    } catch {
      return '';
    }
  }

  const parts = splitPackageId(pkg.id);
  if (!parts) return '';

  const query = new URLSearchParams({ path: icon });
  return (
    [
      '/sd-api/v2/extension/store/file',
      encodeURIComponent(parts.namespace),
      encodeURIComponent(parts.name),
      encodeURIComponent(pkg.version),
    ].join('/') + `?${query.toString()}`
  );
}
