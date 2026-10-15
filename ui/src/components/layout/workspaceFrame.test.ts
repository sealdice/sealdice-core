import { getWorkspaceFrameClass, getWorkspaceSplitClass } from './workspaceFrame.ts';
import { expect, it } from 'vitest';

it('工作区布局模式与内容高度匹配', async () => {
  expect(getWorkspaceFrameClass('fluid')).toBe('workspace-frame');
  expect(getWorkspaceFrameClass('fixed-height')).toBe(
    'workspace-frame workspace-frame--fixed-height'
  );
  expect(getWorkspaceSplitClass('row')).toBe('workspace-split');
  expect(getWorkspaceSplitClass('column')).toBe('workspace-split workspace-split--column');
});
