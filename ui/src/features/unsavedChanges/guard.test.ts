import { expect, it, vi } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';
import { appPinia } from '@/pinia';
import { useUnsavedChangesStore } from './store';
import { setupUnsavedChangesGuard } from './guard';

it('一次确认离开不会跳过下一次未保存检查', async () => {
  const store = useUnsavedChangesStore(appPinia);
  store.clearAllSources();
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/a', '/b', '/c'].map(path => ({ path, component: { render: () => null } })),
  });
  await router.push('/a');
  setupUnsavedChangesGuard(router);
  const confirm = vi.fn().mockResolvedValueOnce(true).mockResolvedValueOnce(false);
  const release = store.setConfirmHandler(confirm);
  try {
    store.registerSource('a', { label: 'A', dirty: true });
    await router.push('/b');
    expect(router.currentRoute.value.path).toBe('/b');
    store.clearAllSources();
    store.registerSource('b', { label: 'B', dirty: true });
    await router.push('/c');
    expect(router.currentRoute.value.path).toBe('/b');
    expect(confirm).toHaveBeenCalledTimes(2);
  } finally {
    release();
    store.clearAllSources();
  }
});
