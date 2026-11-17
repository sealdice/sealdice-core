import { buildPprofBinaryPath, buildPprofTextPath, createPprofEntries } from './model.js';
import { expect, it } from 'vitest';

it('性能采样地址区分文本与二进制导出', async () => {
  const entries = createPprofEntries();
  expect(entries.length).toBe(9);
  expect(entries[0]?.key).toBe('profile');
  expect(entries[1]?.key).toBe('trace');
  expect(entries[2]?.key).toBe('heap');

  expect(buildPprofBinaryPath(entries[0]!, { profileSeconds: 30, traceSeconds: 5 })).toBe(
    '/profile?seconds=30'
  );
  expect(buildPprofBinaryPath(entries[1]!, { profileSeconds: 30, traceSeconds: 5 })).toBe(
    '/trace?seconds=5'
  );
  expect(buildPprofBinaryPath(entries[2]!, { profileSeconds: 30, traceSeconds: 5 })).toBe(
    '/heap?debug=0'
  );
  expect(buildPprofTextPath(entries[2]!)).toBe('/heap?debug=1');
  expect(buildPprofTextPath(entries[0]!)).toBe('');
});
