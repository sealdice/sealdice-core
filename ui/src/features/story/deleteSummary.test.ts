import { describe, expect, it } from 'vitest';
import { summarizeStoryLogs } from './deleteSummary';

describe('summarizeStoryLogs', () => {
  it('includes the count and names of the selected logs', () => {
    expect(
      summarizeStoryLogs([
        { name: '团本 A', groupId: 'group-a' },
        { name: '团本 B', groupId: 'group-b' },
      ])
    ).toBe('共 2 份日志：团本 A（group-a）、团本 B（group-b）');
  });
});
