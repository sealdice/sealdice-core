import { getAppShellContentClass, getAppShellDrawerWidth } from './appShellLayout.ts';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(getAppShellContentClass('default'), 'sd-main-container');
assertEqual(getAppShellContentClass('wide'), 'sd-main-container sd-main-container--wide');
assertEqual(getAppShellDrawerWidth(), 'min(320px, 86vw)');
