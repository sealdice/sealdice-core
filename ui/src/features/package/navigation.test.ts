import { describe, expect, it } from 'vitest';
import { PACKAGE_STORE_ROUTE, resolvePackageManagerTab } from './navigation';

describe('package navigation', () => {
  it('provides a stable store deep link', () => {
    expect(PACKAGE_STORE_ROUTE).toBe('/mod/package?tab=store');
  });

  it('accepts supported tabs and falls back to installed packages', () => {
    expect(resolvePackageManagerTab('store')).toBe('store');
    expect(resolvePackageManagerTab('manage')).toBe('manage');
    expect(resolvePackageManagerTab('installed')).toBe('installed');
    expect(resolvePackageManagerTab('unknown')).toBe('installed');
    expect(resolvePackageManagerTab(['store'])).toBe('installed');
  });
});
