import { storeToRefs } from 'pinia';
import { appPinia } from '@/pinia';
import {
  useUnsavedChangesStore,
  type ActiveUnsavedChangesSource,
  type UnsavedChangesSourceOptions,
  type UnsavedConfirmHandler,
  type UnsavedChangesEvents,
} from './store';

export type {
  ActiveUnsavedChangesSource,
  UnsavedChangesSourceOptions,
  UnsavedConfirmHandler,
  UnsavedChangesEvents,
};

const unsavedChangesStore = useUnsavedChangesStore(appPinia);
const { activeUnsavedChangesSource, hasUnsavedChanges } = storeToRefs(unsavedChangesStore);

// 兼容层：旧代码继续从 state.ts 使用未保存变更 API，实际状态已集中到 Pinia store。
export { activeUnsavedChangesSource, hasUnsavedChanges };
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
