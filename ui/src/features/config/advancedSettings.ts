import { computed, ref } from 'vue';

const advancedSettingsVisible = ref(false);

export const hasAdvancedSettingsAccess = computed(() => advancedSettingsVisible.value);

export function setAdvancedSettingsVisible(value: boolean): void {
  advancedSettingsVisible.value = value;
}
