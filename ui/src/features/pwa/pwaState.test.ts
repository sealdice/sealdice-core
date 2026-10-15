import { isStandaloneDisplayMode, shouldShowPwaInstallEntry } from './pwaState';
import { expect, it } from 'vitest';

it('安装提示根据平台与应用状态派生', async () => {
  expect(isStandaloneDisplayMode(false, false)).toBe(false);
  expect(isStandaloneDisplayMode(true, false)).toBe(true);
  expect(isStandaloneDisplayMode(false, true)).toBe(true);
  expect(isStandaloneDisplayMode(true, true)).toBe(true);

  expect(shouldShowPwaInstallEntry(false, false)).toBe(false);
  expect(shouldShowPwaInstallEntry(true, false)).toBe(true);
  expect(shouldShowPwaInstallEntry(false, true)).toBe(false);
});
