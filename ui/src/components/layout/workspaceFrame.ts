export type WorkspaceFrameMode = 'fluid' | 'fixed-height';

export function getWorkspaceFrameClass(mode: WorkspaceFrameMode): string {
  return mode === 'fixed-height'
    ? 'workspace-frame workspace-frame--fixed-height'
    : 'workspace-frame';
}

export function getWorkspaceSplitClass(direction: 'row' | 'column'): string {
  return direction === 'column' ? 'workspace-split workspace-split--column' : 'workspace-split';
}
