import {
  createEmptyNetworkHealth,
  formatNetworkHealthRelativeTime,
  formatNetworkHealthTimestamp,
  isNetworkHealthTargetOK,
  type NetworkHealthData,
} from './networkHealth.js';
import { expect, it } from 'vitest';

it('网络状态汇总与失败目标展示', async () => {
  const health = (ok: string[], total = 5): NetworkHealthData => ({
    total,
    ok,
    targets: ['baidu', 'seal', 'sign', 'google', 'github'].map(target => ({
      target,
      ok: ok.includes(target),
      durationMs: ok.includes(target) ? 12 : 0,
    })),
    timestamp: 1700000000,
  });

  expect(createEmptyNetworkHealth().timestamp).toBe(0);
  expect(isNetworkHealthTargetOK(health(['seal']), 'seal')).toBe(true);
  expect(isNetworkHealthTargetOK(health(['seal']), 'sign')).toBe(false);
  expect(formatNetworkHealthTimestamp(0)).toBe('');
  expect(formatNetworkHealthTimestamp(1700000000).includes('2023')).toBe(true);
  expect(formatNetworkHealthRelativeTime(0)).toBe('');
});
