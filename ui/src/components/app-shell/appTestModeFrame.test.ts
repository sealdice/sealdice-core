import { getTestModeWatermarkRows } from './appTestModeFrame.ts';
import { expect, it } from 'vitest';

it('展示模式的提示随状态显示', async () => {
  const rows = getTestModeWatermarkRows('仅用于展示，修改无效', 5);
  expect(rows.length).toBe(5);
  expect(rows.every(row => row.includes('仅用于展示，修改无效'))).toBe(true);

  const emptyRows = getTestModeWatermarkRows('', 4);
  expect(emptyRows.length).toBe(0);
});
