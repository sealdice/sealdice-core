import { expect, it } from 'vitest';
import { checkBudget, enforcePrecacheBudget, RESOURCE_BUDGET } from './resourceBudget';

it('预算超限直接失败，不通过提高警告阈值掩盖增长', () => {
  expect(() =>
    checkBudget('chunk', RESOURCE_BUDGET.chunkBytes, RESOURCE_BUDGET.chunkBytes)
  ).not.toThrow();
  expect(() =>
    checkBudget('chunk', RESOURCE_BUDGET.chunkBytes + 1, RESOURCE_BUDGET.chunkBytes)
  ).toThrow('exceeds budget');
});

it('预缓存保留离线业务代码，只去除可选调试控制台', () => {
  const entries = [
    { url: 'assets/vendor-debug-abc.js', size: 500_000 },
    { url: 'assets/reply-abc.js', size: 100_000 },
    { url: 'index.html', size: 1000 },
  ];
  expect(enforcePrecacheBudget(entries).manifest).toEqual(entries.slice(1));
  expect(entries).toHaveLength(3);
  expect(() =>
    enforcePrecacheBudget([{ url: 'app.js', size: RESOURCE_BUDGET.precacheBytes + 1 }])
  ).toThrow('PWA precache');
});
