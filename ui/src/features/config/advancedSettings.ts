import { computed, ref } from 'vue';
import type { AdvancedConfig } from '@/api';

const advancedSettingsVisible = ref(false);

export const hasAdvancedSettingsAccess = computed(() => advancedSettingsVisible.value);

export function setAdvancedSettingsVisible(value: boolean): void {
  advancedSettingsVisible.value = value;
}

function cleanText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

export function normalizeAdvancedConfig(config?: Partial<AdvancedConfig> | null): AdvancedConfig {
  return {
    show: Boolean(config?.show),
    enable: Boolean(config?.enable),
    storyLogBackendUrl: cleanText(config?.storyLogBackendUrl),
    storyLogApiVersion: cleanText(config?.storyLogApiVersion),
    storyLogBackendToken: cleanText(config?.storyLogBackendToken),
  };
}
