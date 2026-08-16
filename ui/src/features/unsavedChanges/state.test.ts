import { it } from 'vitest';
import { appPinia } from '@/pinia';
import { useUnsavedChangesStore } from './store';
import {
  activeUnsavedChangesSource,
  clearUnsavedChangesSource,
  confirmDiscardUnsavedChanges,
  hasUnsavedChanges,
  registerUnsavedChangesSource,
  saveActiveUnsavedChanges,
  setUnsavedChangesConfirmHandler,
} from './state';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) {
      throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
    }
  };

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

  assertEqual(hasUnsavedChanges.value, true);
  assertEqual(activeUnsavedChangesSource.value?.scope, 'test-high');
  assertEqual(await saveActiveUnsavedChanges(), true);
  assertEqual(saved, true);

  setUnsavedChangesConfirmHandler(async source => source.scope === 'test-high');
  assertEqual(await confirmDiscardUnsavedChanges(), true);

  clearUnsavedChangesSource('test-high');
  clearUnsavedChangesSource('test-low');
  assertEqual(hasUnsavedChanges.value, false);
});
