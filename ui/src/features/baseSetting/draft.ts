import { computed, ref } from 'vue';
import type { BaseSettingValueResp } from '@/api';
import {
  cloneBaseSettingValue,
  isBaseSettingDirty,
  normalizeBaseSettingValue,
  type BaseSettingValueModel,
} from './viewModel';

export function useBaseSettingDraft() {
  const currentValue = ref<BaseSettingValueModel | null>(null);
  const initialValue = ref<BaseSettingValueModel | null>(null);

  const dirty = computed(() => {
    if (!currentValue.value || !initialValue.value) return false;
    return isBaseSettingDirty(currentValue.value, initialValue.value);
  });

  function syncRemote(value: BaseSettingValueResp, force = false) {
    if (currentValue.value && dirty.value && !force) return;
    const normalized = normalizeBaseSettingValue(value);
    currentValue.value = cloneBaseSettingValue(normalized);
    initialValue.value = cloneBaseSettingValue(normalized);
  }

  function commitSaved() {
    if (!currentValue.value) return;
    initialValue.value = cloneBaseSettingValue(currentValue.value);
  }

  function resetToRemote() {
    if (!initialValue.value) return;
    currentValue.value = cloneBaseSettingValue(initialValue.value);
  }

  return {
    currentValue,
    initialValue,
    dirty,
    syncRemote,
    commitSaved,
    resetToRemote,
  };
}
