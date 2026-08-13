import { it } from 'vitest';
import { formatAppChannel } from './appChannel.js';

it('formats supported app channels consistently', () => {
  const assertEqual = (actual: string, expected: string) => {
    if (actual !== expected) throw new Error(`expected ${expected}, got ${actual}`);
  };

  assertEqual(formatAppChannel('stable'), '正式版');
  assertEqual(formatAppChannel('dev'), '开发版');
  assertEqual(formatAppChannel('nightly'), '未知');
  assertEqual(formatAppChannel(undefined), '未知');
});
