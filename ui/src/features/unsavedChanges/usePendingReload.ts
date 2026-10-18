import type { MaybeRefOrGetter } from 'vue';
import { useUnsavedChanges } from './useUnsavedChanges';

export interface PendingReloadOptions {
  label: MaybeRefOrGetter<string>;
  /** 是否存在「已保存但未生效」的改动。 */
  pending: MaybeRefOrGetter<boolean>;
  reload: () => Promise<unknown> | unknown;
  reloading?: MaybeRefOrGetter<boolean>;
  canReload?: MaybeRefOrGetter<boolean>;
  actionText?: MaybeRefOrGetter<string>;
  priority?: number;
}

/**
 * 注册「需要重载才生效」状态。与 useUnsavedChanges 共用同一套载体，
 * 但不参与路由守卫 —— 离开页面不会丢失已保存的改动。
 */
export function usePendingReload(scope: string, options: PendingReloadOptions) {
  return useUnsavedChanges(scope, {
    kind: 'reload',
    label: options.label,
    dirty: options.pending,
    save: options.reload,
    saving: options.reloading,
    canSave: options.canReload,
    actionText: options.actionText,
    priority: options.priority,
  });
}
