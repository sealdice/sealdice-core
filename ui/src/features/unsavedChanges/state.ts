import mitt from 'mitt';
import { computed, ref, toValue, type MaybeRefOrGetter } from 'vue';

export interface UnsavedChangesSourceOptions {
  label: MaybeRefOrGetter<string>;
  dirty: MaybeRefOrGetter<boolean>;
  save?: () => Promise<unknown> | unknown;
  saving?: MaybeRefOrGetter<boolean>;
  canSave?: MaybeRefOrGetter<boolean>;
  confirmMessage?: MaybeRefOrGetter<string>;
  priority?: number;
}

interface RegisteredUnsavedChangesSource extends UnsavedChangesSourceOptions {
  scope: string;
  order: number;
}

export interface ActiveUnsavedChangesSource {
  scope: string;
  label: string;
  save?: () => Promise<unknown> | unknown;
  saving: boolean;
  canSave: boolean;
  confirmMessage: string;
  priority: number;
}

type UnsavedConfirmHandler = (source: ActiveUnsavedChangesSource) => Promise<boolean>;
type UnsavedChangesEvents = {
  changed: void;
};

const registeredSources = ref<Record<string, RegisteredUnsavedChangesSource>>({});
const confirmHandler = ref<UnsavedConfirmHandler | null>(null);
export const unsavedChangesEmitter = mitt<UnsavedChangesEvents>();

let registerOrder = 0;

function resolveSource(source: RegisteredUnsavedChangesSource): ActiveUnsavedChangesSource | null {
  if (!toValue(source.dirty)) return null;

  const label = toValue(source.label).trim() || '当前页面';
  const confirmMessage = source.confirmMessage
    ? toValue(source.confirmMessage)
    : `${label} 还有修改，确定要忽略？`;

  return {
    scope: source.scope,
    label,
    save: source.save,
    saving: source.saving ? Boolean(toValue(source.saving)) : false,
    canSave: source.canSave ? Boolean(toValue(source.canSave)) : Boolean(source.save),
    confirmMessage,
    priority: source.priority ?? 0,
  };
}

export const activeUnsavedChangesSource = computed<ActiveUnsavedChangesSource | null>(() => {
  const candidates = Object.values(registeredSources.value)
    .map(resolveSource)
    .filter((source): source is ActiveUnsavedChangesSource => source !== null);

  if (!candidates.length) return null;

  return candidates.sort((left, right) => {
    if (right.priority !== left.priority) return right.priority - left.priority;
    const leftOrder = registeredSources.value[left.scope]?.order ?? 0;
    const rightOrder = registeredSources.value[right.scope]?.order ?? 0;
    return rightOrder - leftOrder;
  })[0]!;
});

export const hasUnsavedChanges = computed(() => activeUnsavedChangesSource.value !== null);

export function registerUnsavedChangesSource(scope: string, options: UnsavedChangesSourceOptions): void {
  registeredSources.value = {
    ...registeredSources.value,
    [scope]: {
      ...options,
      scope,
      order: registerOrder++,
    },
  };
  unsavedChangesEmitter.emit('changed');
}

export function clearUnsavedChangesSource(scope: string): void {
  if (!(scope in registeredSources.value)) return;

  const nextSources = { ...registeredSources.value };
  delete nextSources[scope];
  registeredSources.value = nextSources;
  unsavedChangesEmitter.emit('changed');
}

export function setUnsavedChangesConfirmHandler(handler: UnsavedConfirmHandler | null): void {
  confirmHandler.value = handler;
}

export async function confirmDiscardUnsavedChanges(): Promise<boolean> {
  const source = activeUnsavedChangesSource.value;
  if (!source) return true;

  if (confirmHandler.value) {
    return confirmHandler.value(source);
  }

  if (typeof window === 'undefined') return true;
  return window.confirm(source.confirmMessage);
}

export async function saveActiveUnsavedChanges(): Promise<boolean> {
  const source = activeUnsavedChangesSource.value;
  if (!source?.save) return false;

  try {
    await source.save();
    return true;
  } catch {
    return false;
  }
}
