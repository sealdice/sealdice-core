import type { Router } from 'vue-router';
import { appPinia } from '@/pinia';
import { useUnsavedChangesStore } from './store';

export function setupUnsavedChangesGuard(router: Router): void {
  router.beforeEach(async (to, from) => {
    if (to.fullPath === from.fullPath) return true;
    const store = useUnsavedChangesStore(appPinia);
    return !store.hasUnsavedChanges || (await store.confirmDiscard());
  });
}
