import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';
import { useUnsavedChangesStore } from './store';
import { storeToRefs } from 'pinia';
const store = useUnsavedChangesStore(appPinia);
const { activeUnsavedChangesSource, hasUnsavedChanges } = storeToRefs(store);
const {
  clearSource: clearUnsavedChangesSource,
  confirmDiscard: confirmDiscardUnsavedChanges,
  registerSource: registerUnsavedChangesSource,
  runPendingAction,
  setConfirmHandler: setUnsavedChangesConfirmHandler,
} = store;

it('未保存来源注册、保存与离开确认', async () => {
  const store = useUnsavedChangesStore(appPinia);
  store.clearAllSources();
  setUnsavedChangesConfirmHandler(null);

  let saved = false;
  registerUnsavedChangesSource('test-low', {
    label: '低优先级',
    dirty: true,
    priority: 1,
  });
  registerUnsavedChangesSource('test-high', {
    label: '高优先级',
    dirty: true,
    priority: 10,
    save: () => {
      saved = true;
    },
  });

  expect(hasUnsavedChanges.value).toBe(true);
  expect(activeUnsavedChangesSource.value?.scope).toBe('test-high');
  expect(await runPendingAction('test-high')).toBe(true);
  expect(saved).toBe(true);

  setUnsavedChangesConfirmHandler(async source => source.scope === 'test-high');
  expect(await confirmDiscardUnsavedChanges()).toBe(true);

  clearUnsavedChangesSource('test-high');
  clearUnsavedChangesSource('test-low');
  expect(hasUnsavedChanges.value).toBe(false);
});
