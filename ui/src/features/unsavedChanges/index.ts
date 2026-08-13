export {
  activePendingActions,
  activeUnsavedChangesSource,
  hasPendingActions,
  discardPendingAction,
  hasUnsavedChanges,
  runPendingAction,
  saveActiveUnsavedChanges,
  setUnsavedChangesConfirmHandler,
} from './state';
export { setupUnsavedChangesGuard } from './guard';
export { useUnsavedChanges } from './useUnsavedChanges';
export { usePendingReload } from './usePendingReload';
export type { PendingReloadOptions } from './usePendingReload';
export { isPendingActionAnchorVisible, setPendingActionAnchor } from './usePendingActionAnchor';
export type {
  ActiveUnsavedChangesSource,
  PendingActionKind,
  UnsavedChangesSourceOptions,
} from './state';
