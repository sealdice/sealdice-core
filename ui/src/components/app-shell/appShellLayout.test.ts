import {
  APP_SHELL_DESKTOP_BREAKPOINT,
  APP_SHELL_MOBILE_MAX_WIDTH,
  getAppShellContainerClass,
  getAppShellContentClass,
  getAppShellDrawerWidth,
  isAppShellMobileWidth,
} from './appShellLayout.ts';
import { it } from 'vitest';

it('passes', async () => {

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(getAppShellContentClass('default'), 'sd-main-container');
assertEqual(getAppShellContentClass('wide'), 'sd-main-container sd-main-container--wide');
assertEqual(getAppShellContainerClass('default'), 'sd-page-shell');
assertEqual(getAppShellContainerClass('workspace'), 'sd-page-shell sd-page-shell--workspace');
assertEqual(getAppShellDrawerWidth(), 'min(200px, 76vw)');
assertEqual(APP_SHELL_DESKTOP_BREAKPOINT, 'md');
assertEqual(APP_SHELL_MOBILE_MAX_WIDTH, 767.9);
assertEqual(isAppShellMobileWidth(640), true);
assertEqual(isAppShellMobileWidth(767), true);
assertEqual(isAppShellMobileWidth(768), false);
});
