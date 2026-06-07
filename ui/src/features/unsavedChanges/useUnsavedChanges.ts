import { onBeforeUnmount } from 'vue';
import {
  clearUnsavedChangesSource,
  registerUnsavedChangesSource,
  type UnsavedChangesSourceOptions,
} from './state';

export function useUnsavedChanges(scope: string, options: UnsavedChangesSourceOptions) {
  registerUnsavedChangesSource(scope, options);

  onBeforeUnmount(() => {
    clearUnsavedChangesSource(scope);
  });

  return {
    clear() {
      clearUnsavedChangesSource(scope);
    },
  };
}
