import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';
import { useUnsavedChangesStore } from './store';
import {
  activePendingActions,
  activeUnsavedChangesSource,
  confirmDiscardUnsavedChanges,
  discardPendingAction,
  hasPendingActions,
  hasUnsavedChanges,
  registerUnsavedChangesSource,
  runPendingAction,
  setUnsavedChangesConfirmHandler,
} from './state';

function resetStore() {
  const store = useUnsavedChangesStore(appPinia);
  store.clearAllSources();
  setUnsavedChangesConfirmHandler(null);
  return store;
}

it('未保存与待重载可以同时成立并并列展示', () => {
  resetStore();

  registerUnsavedChangesSource('cfg', { label: '拦截设置', dirty: true, save: () => {} });
  registerUnsavedChangesSource('reload', {
    kind: 'reload',
    label: '拦截配置',
    dirty: true,
    save: () => {},
  });

  expect(activePendingActions.value.map(item => item.scope).sort()).toEqual(['cfg', 'reload']);
  expect(hasPendingActions.value).toBe(true);
});

it('路由守卫只由未保存驱动，待重载不阻止离开', async () => {
  resetStore();

  registerUnsavedChangesSource('reload-only', {
    kind: 'reload',
    label: '帮助文档',
    dirty: true,
    save: () => {},
  });

  // 只有待重载时：有待处理状态，但没有未保存，离开页面不该被拦。
  expect(hasPendingActions.value).toBe(true);
  expect(hasUnsavedChanges.value).toBe(false);
  expect(activeUnsavedChangesSource.value).toBeNull();
  expect(await confirmDiscardUnsavedChanges()).toBe(true);
});

it('按 scope 执行对应的待处理动作', async () => {
  resetStore();

  const calls: string[] = [];
  registerUnsavedChangesSource('a', { label: 'A', dirty: true, save: () => calls.push('a') });
  registerUnsavedChangesSource('b', {
    kind: 'reload',
    label: 'B',
    dirty: true,
    save: () => calls.push('b'),
  });

  expect(await runPendingAction('b')).toBe(true);
  expect(calls).toEqual(['b']);

  expect(await runPendingAction('missing')).toBe(false);
});

it('仅在来源提供 discard 时可放弃改动', async () => {
  resetStore();

  let discarded = false;
  registerUnsavedChangesSource('with-discard', {
    label: 'A',
    dirty: true,
    save: () => {},
    discard: () => {
      discarded = true;
    },
  });
  registerUnsavedChangesSource('without-discard', { label: 'B', dirty: true, save: () => {} });

  const byScope = new Map(activePendingActions.value.map(item => [item.scope, item]));
  expect(typeof byScope.get('with-discard')?.discard).toBe('function');
  expect(byScope.get('without-discard')?.discard).toBeUndefined();

  expect(await discardPendingAction('with-discard')).toBe(true);
  expect(discarded).toBe(true);
  expect(await discardPendingAction('without-discard')).toBe(false);
});

it('操作文案按类型给默认值，可被覆盖', () => {
  resetStore();

  registerUnsavedChangesSource('unsaved', { label: 'A', dirty: true, save: () => {} });
  registerUnsavedChangesSource('reload', {
    kind: 'reload',
    label: 'B',
    dirty: true,
    save: () => {},
  });
  registerUnsavedChangesSource('custom', {
    kind: 'reload',
    label: 'C',
    dirty: true,
    save: () => {},
    actionText: '重载拦截',
  });

  const byScope = new Map(activePendingActions.value.map(item => [item.scope, item]));
  expect(byScope.get('unsaved')?.actionText).toBe('保存设置');
  expect(byScope.get('reload')?.actionText).toBe('立即重载');
  expect(byScope.get('custom')?.actionText).toBe('重载拦截');
});

it('dirty 为 false 的来源不出现在待处理列表中', () => {
  resetStore();

  registerUnsavedChangesSource('clean', { label: 'A', dirty: false, save: () => {} });
  expect(activePendingActions.value).toEqual([]);
  expect(hasPendingActions.value).toBe(false);
});
