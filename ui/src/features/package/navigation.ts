export const PACKAGE_STORE_ROUTE = '/mod/package?tab=store';

export type PackageManagerTab = 'installed' | 'store' | 'manage';

export function resolvePackageManagerTab(value: unknown): PackageManagerTab {
  return value === 'store' || value === 'manage' ? value : 'installed';
}
