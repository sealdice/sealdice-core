import { it } from 'vitest';
import { formatRuntimeMode, formatRuntimeSummary } from './runtimeSummary.js';

const assertEqual = (actual: string, expected: string) => {
  if (actual !== expected) throw new Error(`expected ${expected}, got ${actual}`);
};

it('joins runtime segments with the fullwidth separator', () => {
  const runtime = { OS: 'linux', arch: 'amd64', containerMode: false, justForTest: false };

  assertEqual(formatRuntimeSummary(runtime), 'linux・amd64');
  assertEqual(formatRuntimeSummary(runtime, { withMode: true }), 'linux・amd64');
});

it('appends the runtime mode only when it is not a plain local run', () => {
  assertEqual(
    formatRuntimeSummary(
      { OS: 'linux', arch: 'amd64', containerMode: true, justForTest: false },
      { withMode: true }
    ),
    'linux・amd64・容器模式'
  );
  assertEqual(
    formatRuntimeSummary(
      { OS: 'linux', arch: 'amd64', containerMode: false, justForTest: true },
      { withMode: true }
    ),
    'linux・amd64・展示模式'
  );
  assertEqual(
    formatRuntimeSummary({ OS: 'linux', arch: 'amd64', containerMode: true }),
    'linux・amd64'
  );
});

it('falls back when the runtime is unavailable or incomplete', () => {
  assertEqual(formatRuntimeSummary(undefined), '读取中');
  assertEqual(formatRuntimeSummary({ OS: 'linux' }), '读取中');
  assertEqual(formatRuntimeSummary({ arch: 'amd64' }), '读取中');
});

it('formats the runtime mode field', () => {
  assertEqual(formatRuntimeMode(undefined), '读取中');
  assertEqual(formatRuntimeMode({ justForTest: true, containerMode: true }), '展示模式');
  assertEqual(formatRuntimeMode({ containerMode: true }), '容器模式');
  assertEqual(formatRuntimeMode({ OS: 'linux', arch: 'amd64' }), '本机运行');
});
