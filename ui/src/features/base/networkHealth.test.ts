import {
  createEmptyNetworkHealth,
  formatNetworkHealthRelativeTime,
  formatNetworkHealthTimestamp,
  isNetworkHealthTargetOK,
  type NetworkHealthData,
} from './networkHealth.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

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

assertEqual(createEmptyNetworkHealth().timestamp, 0);
assertEqual(isNetworkHealthTargetOK(health(['seal']), 'seal'), true);
assertEqual(isNetworkHealthTargetOK(health(['seal']), 'sign'), false);
assertEqual(formatNetworkHealthTimestamp(0), '');
assertEqual(formatNetworkHealthTimestamp(1700000000).includes('2023'), true);
assertEqual(formatNetworkHealthRelativeTime(0), '');
