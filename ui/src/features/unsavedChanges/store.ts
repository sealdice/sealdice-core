import mitt from 'mitt';
import { defineStore } from 'pinia';
import { computed, markRaw, ref, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';

/**
 * 待处理状态分两类，视觉载体相同，语义不同：
 * - unsaved：有改动尚未保存，离开页面会丢数据，因此参与路由守卫。
 * - reload：已保存但需要重载才生效，离开页面无损失，不参与路由守卫。
 * 两者可以同时成立，载体并列展示。
 */
export type PendingActionKind = 'unsaved' | 'reload';

export interface UnsavedChangesSourceOptions {
  label: MaybeRefOrGetter<string>;
  dirty: MaybeRefOrGetter<boolean>;
  save?: () => Promise<unknown> | unknown;
  saving?: MaybeRefOrGetter<boolean>;
  canSave?: MaybeRefOrGetter<boolean>;
  confirmMessage?: MaybeRefOrGetter<string>;
  priority?: number;
  /** 默认 unsaved，保持既有调用方零改动。 */
  kind?: PendingActionKind;
  /** 操作按钮文案，默认按 kind 推导（保存设置 / 立即重载）。 */
  actionText?: MaybeRefOrGetter<string>;
}

interface RegisteredUnsavedChangesSource extends UnsavedChangesSourceOptions {
  scope: string;
  order: number;
}

export interface ActiveUnsavedChangesSource {
  scope: string;
  kind: PendingActionKind;
  label: string;
  actionText: string;
  save?: () => Promise<unknown> | unknown;
  saving: boolean;
  canSave: boolean;
  confirmMessage: string;
  priority: number;
}

export type UnsavedConfirmHandler = (source: ActiveUnsavedChangesSource) => Promise<boolean>;
export type UnsavedChangesEvents = {
  changed: void;
};

const DEFAULT_ACTION_TEXT: Record<PendingActionKind, string> = {
  unsaved: '保存设置',
  reload: '立即重载',
};

export const useUnsavedChangesStore = defineStore('unsaved-changes', () => {
  const registeredSources = ref<Record<string, RegisteredUnsavedChangesSource>>({});
  const confirmHandler = shallowRef<UnsavedConfirmHandler | null>(null);
  const unsavedChangesEmitter = markRaw(mitt<UnsavedChangesEvents>());

  let registerOrder = 0;

  function resolveSource(source: RegisteredUnsavedChangesSource): ActiveUnsavedChangesSource | null {
    if (!toValue(source.dirty)) return null;

    const kind = source.kind ?? 'unsaved';
    const label = toValue(source.label).trim() || '当前页面';
    const confirmMessage = source.confirmMessage
      ? toValue(source.confirmMessage)
      : `${label} 还有修改，确定要忽略？`;
    const actionText = source.actionText
      ? toValue(source.actionText).trim() || DEFAULT_ACTION_TEXT[kind]
      : DEFAULT_ACTION_TEXT[kind];

    return {
      scope: source.scope,
      kind,
      label,
      actionText,
      save: source.save,
      saving: source.saving ? Boolean(toValue(source.saving)) : false,
      canSave: source.canSave ? Boolean(toValue(source.canSave)) : Boolean(source.save),
      confirmMessage,
      priority: source.priority ?? 0,
    };
  }

  function sortByPriority(
    left: ActiveUnsavedChangesSource,
    right: ActiveUnsavedChangesSource
  ): number {
    if (right.priority !== left.priority) return right.priority - left.priority;
    const leftOrder = registeredSources.value[left.scope]?.order ?? 0;
    const rightOrder = registeredSources.value[right.scope]?.order ?? 0;
    return rightOrder - leftOrder;
  }

  /** 当前成立的全部待处理状态，已按优先级排序。 */
  const activePendingActions = computed<ActiveUnsavedChangesSource[]>(() =>
    Object.values(registeredSources.value)
      .map(resolveSource)
      .filter((source): source is ActiveUnsavedChangesSource => source !== null)
      .sort(sortByPriority)
  );

  /** 仅未保存类，路由守卫与 beforeunload 只看这一类。 */
  const activeUnsavedChangesSource = computed<ActiveUnsavedChangesSource | null>(
    () => activePendingActions.value.find(source => source.kind === 'unsaved') ?? null
  );

  const hasUnsavedChanges = computed(() => activeUnsavedChangesSource.value !== null);
  const hasPendingActions = computed(() => activePendingActions.value.length > 0);

  function emitChanged(): void {
    unsavedChangesEmitter.emit('changed');
  }

  function registerSource(scope: string, options: UnsavedChangesSourceOptions): void {
    // 每个业务区用 scope 注册自己的脏状态，Pinia 只负责选择当前最高优先级来源。
    registeredSources.value = {
      ...registeredSources.value,
      [scope]: {
        ...options,
        scope,
        order: registerOrder++,
      },
    };
    emitChanged();
  }

  function clearSource(scope: string): void {
    if (!(scope in registeredSources.value)) return;

    const nextSources = { ...registeredSources.value };
    delete nextSources[scope];
    registeredSources.value = nextSources;
    emitChanged();
  }

  function clearAllSources(): void {
    registeredSources.value = {};
    emitChanged();
  }

  function setConfirmHandler(handler: UnsavedConfirmHandler | null): void {
    confirmHandler.value = handler;
  }

  async function confirmDiscard(): Promise<boolean> {
    const source = activeUnsavedChangesSource.value;
    if (!source) return true;

    if (confirmHandler.value) {
      return confirmHandler.value(source);
    }

    if (typeof window === 'undefined') return true;
    return window.confirm(source.confirmMessage);
  }

  async function runPendingAction(scope: string): Promise<boolean> {
    const source = activePendingActions.value.find(item => item.scope === scope);
    if (!source?.save) return false;

    try {
      await source.save();
      return true;
    } catch {
      return false;
    }
  }

  async function saveActive(): Promise<boolean> {
    const source = activeUnsavedChangesSource.value;
    if (!source) return false;
    return runPendingAction(source.scope);
  }

  return {
    activePendingActions,
    activeUnsavedChangesSource,
    hasUnsavedChanges,
    hasPendingActions,
    unsavedChangesEmitter,
    registerSource,
    clearSource,
    clearAllSources,
    setConfirmHandler,
    confirmDiscard,
    runPendingAction,
    saveActive,
  };
});
