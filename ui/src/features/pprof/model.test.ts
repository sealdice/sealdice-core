import {
  buildPprofBinaryPath,
  buildPprofTextPath,
  createPprofEntries,
} from './model.js';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const entries = createPprofEntries();
  assertEqual(entries.length, 9);
  assertEqual(entries[0]?.key, 'profile');
  assertEqual(entries[1]?.key, 'trace');
  assertEqual(entries[2]?.key, 'heap');

  assertEqual(buildPprofBinaryPath(entries[0]!, { profileSeconds: 30, traceSeconds: 5 }), '/profile?seconds=30');
  assertEqual(buildPprofBinaryPath(entries[1]!, { profileSeconds: 30, traceSeconds: 5 }), '/trace?seconds=5');
  assertEqual(buildPprofBinaryPath(entries[2]!, { profileSeconds: 30, traceSeconds: 5 }), '/heap?debug=0');
  assertEqual(buildPprofTextPath(entries[2]!), '/heap?debug=1');
  assertEqual(buildPprofTextPath(entries[0]!), '');
});
