import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import {
  useUnsavedChangesStore,
  type ActiveUnsavedChangesSource,
  type PendingActionKind,
  type UnsavedChangesSourceOptions,
  type UnsavedConfirmHandler,
  type UnsavedChangesEvents,
} from './store';

export type {
  ActiveUnsavedChangesSource,
  PendingActionKind,
  UnsavedChangesSourceOptions,
  UnsavedConfirmHandler,
  UnsavedChangesEvents,
};

const unsavedChangesStore = useUnsavedChangesStore(appPinia);
const { activePendingActions, activeUnsavedChangesSource, hasUnsavedChanges, hasPendingActions } =
  storeToRefs(unsavedChangesStore);

// 兼容层：旧代码继续从 state.ts 使用未保存变更 API，实际状态已集中到 Pinia store。
export {
  activePendingActions,
  activeUnsavedChangesSource,
  hasUnsavedChanges,
  hasPendingActions,
};
export const unsavedChangesEmitter = unsavedChangesStore.unsavedChangesEmitter;

export function registerUnsavedChangesSource(scope: string, options: UnsavedChangesSourceOptions): void {
  unsavedChangesStore.registerSource(scope, options);
}

export function clearUnsavedChangesSource(scope: string): void {
  unsavedChangesStore.clearSource(scope);
}

export function setUnsavedChangesConfirmHandler(handler: UnsavedConfirmHandler | null): void {
  unsavedChangesStore.setConfirmHandler(handler);
}

export async function confirmDiscardUnsavedChanges(): Promise<boolean> {
  return unsavedChangesStore.confirmDiscard();
}

export async function saveActiveUnsavedChanges(): Promise<boolean> {
  return unsavedChangesStore.saveActive();
}

export async function runPendingAction(scope: string): Promise<boolean> {
  return unsavedChangesStore.runPendingAction(scope);
}
