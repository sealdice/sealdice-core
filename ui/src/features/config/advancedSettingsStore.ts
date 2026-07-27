import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';

export const useAdvancedSettingsStore = defineStore('advanced-settings', () => {
  const advancedSettingsVisible = shallowRef(false);
  const hasAdvancedSettingsAccess = computed(() => advancedSettingsVisible.value);

  function setAdvancedSettingsVisible(value: boolean): void {
    advancedSettingsVisible.value = value;
  }

  return {
    advancedSettingsVisible,
    hasAdvancedSettingsAccess,
    setAdvancedSettingsVisible,
  };
});
