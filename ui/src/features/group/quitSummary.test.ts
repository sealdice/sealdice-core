import { describe, expect, it } from 'vitest';
import { formatGroupQuitTarget } from './quitSummary';

describe('formatGroupQuitTarget', () => {
  it('includes the group name and account for a single quit', () => {
    expect(formatGroupQuitTarget({ groupId: 'group-1', groupName: '测试群', diceId: 'QQ:1' })).toBe(
      'group-1「测试群」，账号 QQ:1',
    );
  });

  it('falls back to the group id when the name is unavailable', () => {
    expect(formatGroupQuitTarget({ groupId: 'group-2', groupName: '  ' })).toBe('group-2');
  });
});
