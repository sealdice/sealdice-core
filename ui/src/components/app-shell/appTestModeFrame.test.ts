import { getTestModeWatermarkRows } from './appTestModeFrame.ts';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const rows = getTestModeWatermarkRows('仅用于展示，修改无效', 5);
  assertEqual(rows.length, 5);
  assertEqual(
    rows.every(row => row.includes('仅用于展示，修改无效')),
    true
  );

  const emptyRows = getTestModeWatermarkRows('', 4);
  assertEqual(emptyRows.length, 0);
});
