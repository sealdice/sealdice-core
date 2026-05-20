import type { Router } from 'vue-router';
import { confirmDiscardUnsavedChanges, hasUnsavedChanges } from './state';

let bypassNextUnsavedGuard = false;

export function setupUnsavedChangesGuard(router: Router): void {
  router.beforeEach(async (to, from) => {
    if (bypassNextUnsavedGuard) {
      bypassNextUnsavedGuard = false;
      return true;
    }

    if (to.fullPath === from.fullPath) return true;
    if (!hasUnsavedChanges.value) return true;

    const confirmed = await confirmDiscardUnsavedChanges();
    if (!confirmed) return false;

    bypassNextUnsavedGuard = true;
    return true;
  });
}
