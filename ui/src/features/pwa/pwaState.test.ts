import { isStandaloneDisplayMode } from './pwaState';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(isStandaloneDisplayMode(false, false), false);
assertEqual(isStandaloneDisplayMode(true, false), true);
assertEqual(isStandaloneDisplayMode(false, true), true);
assertEqual(isStandaloneDisplayMode(true, true), true);

