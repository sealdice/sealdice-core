import { getWorkspaceFrameClass, getWorkspaceSplitClass } from './workspaceFrame.ts';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  assertEqual(getWorkspaceFrameClass('fluid'), 'workspace-frame');
  assertEqual(
    getWorkspaceFrameClass('fixed-height'),
    'workspace-frame workspace-frame--fixed-height'
  );
  assertEqual(getWorkspaceSplitClass('row'), 'workspace-split');
  assertEqual(getWorkspaceSplitClass('column'), 'workspace-split workspace-split--column');
});
