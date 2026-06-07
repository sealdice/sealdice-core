import { computed, ref } from 'vue';
import {
  buildHelpdocConfigPayload,
  cloneHelpdocAliases,
  isHelpdocConfigDirty,
  normalizeHelpdocAliases,
} from './viewModel';

export function useHelpdocConfigDraft() {
  const currentAliases = ref(new Map<string, string[]>());
  const initialAliases = ref(new Map<string, string[]>());
  const ready = ref(false);

  const dirty = computed(() => isHelpdocConfigDirty(currentAliases.value, initialAliases.value));

  function syncRemote(aliases: Record<string, string[] | null | undefined>, force = false) {
    if (ready.value && dirty.value && !force) return;
    const normalized = normalizeHelpdocAliases(aliases);
    currentAliases.value = cloneHelpdocAliases(normalized);
    initialAliases.value = cloneHelpdocAliases(normalized);
    ready.value = true;
  }

  function addAlias(groupKey: string, alias: string) {
    const value = alias.trim();
    if (!value) return false;
    for (const aliases of currentAliases.value.values()) {
      if (aliases.includes(value)) return false;
    }
    const next = cloneHelpdocAliases(currentAliases.value);
    next.set(groupKey, [...(next.get(groupKey) ?? []), value]);
    currentAliases.value = next;
    return true;
  }

  function removeAlias(groupKey: string, alias: string) {
    const next = cloneHelpdocAliases(currentAliases.value);
    next.set(groupKey, (next.get(groupKey) ?? []).filter(item => item !== alias));
    currentAliases.value = next;
  }

  function commitSaved() {
    initialAliases.value = cloneHelpdocAliases(currentAliases.value);
  }

  function resetToRemote() {
    currentAliases.value = cloneHelpdocAliases(initialAliases.value);
  }

  return {
    currentAliases,
    initialAliases,
    dirty,
    syncRemote,
    addAlias,
    removeAlias,
    commitSaved,
    resetToRemote,
    payload: computed(() => buildHelpdocConfigPayload(currentAliases.value)),
  };
}
