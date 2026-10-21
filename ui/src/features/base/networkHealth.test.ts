import {
  createEmptyNetworkHealth,
  formatNetworkHealthRelativeTime,
  formatNetworkHealthTimestamp,
  isNetworkHealthTargetOK,
  normalizeNetworkHealthData,
  type NetworkHealthData,
} from './networkHealth.js';
import { expect, it } from 'vitest';

it('网络状态汇总与失败目标展示', async () => {
  const health = (ok: string[], total = 6): NetworkHealthData => ({
    total,
    ok,
    targets: ['qq', 'kook', 'discord', 'telegram', 'dingtalk', 'slack'].map(target => ({
      target,
      name: target,
      label: `${target} 机器人 API`,
      ok: ok.includes(target),
      durationMs: ok.includes(target) ? 12 : 0,
    })),
    timestamp: 1700000000,
  });

  expect(createEmptyNetworkHealth().timestamp).toBe(0);
  expect(isNetworkHealthTargetOK(health(['qq']), 'qq')).toBe(true);
  expect(isNetworkHealthTargetOK(health(['qq']), 'slack')).toBe(false);
  expect(formatNetworkHealthTimestamp(0)).toBe('');
  expect(formatNetworkHealthTimestamp(1700000000).includes('2023')).toBe(true);
  expect(formatNetworkHealthRelativeTime(0)).toBe('');
});

it('保留接口返回的目标名称与标签', () => {
  const data = normalizeNetworkHealthData({
    total: 1,
    ok: ['kook'],
    targets: [
      {
        target: 'kook',
        name: 'KOOK',
        label: 'KOOK 机器人 API',
        ok: true,
        durationMs: 8,
      },
    ],
    timestamp: 1700000000,
  });

  expect(data.targets[0]).toEqual({
    target: 'kook',
    name: 'KOOK',
    label: 'KOOK 机器人 API',
    ok: true,
    durationMs: 8,
  });
});
