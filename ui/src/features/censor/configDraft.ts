import { computed, ref } from 'vue';
import type { CensorConfigBody } from '@/api';
import {
  cloneCensorConfig,
  createDefaultCensorConfig,
  isCensorConfigDirty,
} from './viewModel';

export function useCensorConfigDraft() {
  const currentConfig = ref<CensorConfigBody>(createDefaultCensorConfig());
  const initialConfig = ref<CensorConfigBody>(createDefaultCensorConfig());
  const ready = ref(false);

  const dirty = computed(() => isCensorConfigDirty(currentConfig.value, initialConfig.value));

  function syncRemote(config: CensorConfigBody, force = false) {
    if (ready.value && dirty.value && !force) return;
    currentConfig.value = cloneCensorConfig(config);
    initialConfig.value = cloneCensorConfig(config);
    ready.value = true;
  }

  function commitSaved() {
    initialConfig.value = cloneCensorConfig(currentConfig.value);
  }

  function resetToRemote() {
    currentConfig.value = cloneCensorConfig(initialConfig.value);
  }

  return {
    currentConfig,
    initialConfig,
    dirty,
    syncRemote,
    commitSaved,
    resetToRemote,
  };
}
