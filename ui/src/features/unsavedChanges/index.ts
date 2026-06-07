export {
  activeUnsavedChangesSource,
  hasUnsavedChanges,
  saveActiveUnsavedChanges,
  setUnsavedChangesConfirmHandler,
} from './state';
export { setupUnsavedChangesGuard } from './guard';
export { useUnsavedChanges } from './useUnsavedChanges';
export type { ActiveUnsavedChangesSource, UnsavedChangesSourceOptions } from './state';
