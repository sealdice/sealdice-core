import { onBeforeUnmount } from 'vue';
import { useUnsavedChangesStore, type UnsavedChangesSourceOptions } from './store';

export function useUnsavedChanges(scope: string, options: UnsavedChangesSourceOptions) {
  const store = useUnsavedChangesStore();
  store.registerSource(scope, options);
  const clear = () => store.clearSource(scope);
  onBeforeUnmount(clear);
  return { clear };
}
