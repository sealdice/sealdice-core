import { expect, it } from 'vitest';
import { formatRuntimeMode, formatRuntimeSummary } from './runtimeSummary.js';

it('joins runtime segments with the fullwidth separator', () => {
  const runtime = { OS: 'linux', arch: 'amd64', containerMode: false, justForTest: false };

  expect(formatRuntimeSummary(runtime)).toBe('linux・amd64');
  expect(formatRuntimeSummary(runtime, { withMode: true })).toBe('linux・amd64');
});

it('appends the runtime mode only when it is not a plain local run', () => {
  expect(
    formatRuntimeSummary(
      { OS: 'linux', arch: 'amd64', containerMode: true, justForTest: false },
      { withMode: true }
    )
  ).toBe('linux・amd64・容器模式');
  expect(
    formatRuntimeSummary(
      { OS: 'linux', arch: 'amd64', containerMode: false, justForTest: true },
      { withMode: true }
    )
  ).toBe('linux・amd64・展示模式');
  expect(formatRuntimeSummary({ OS: 'linux', arch: 'amd64', containerMode: true })).toBe(
    'linux・amd64'
  );
});

it('falls back when the runtime is unavailable or incomplete', () => {
  expect(formatRuntimeSummary(undefined)).toBe('读取中');
  expect(formatRuntimeSummary({ OS: 'linux' })).toBe('读取中');
  expect(formatRuntimeSummary({ arch: 'amd64' })).toBe('读取中');
});

it('formats the runtime mode field', () => {
  expect(formatRuntimeMode(undefined)).toBe('读取中');
  expect(formatRuntimeMode({ justForTest: true, containerMode: true })).toBe('展示模式');
  expect(formatRuntimeMode({ containerMode: true })).toBe('容器模式');
  expect(formatRuntimeMode({ OS: 'linux', arch: 'amd64' })).toBe('本机运行');
});
