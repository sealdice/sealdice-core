import { storeToRefs } from 'pinia';
import type { AdvancedConfig } from '@/api';
import { appPinia } from '@/pinia';
import { useAdvancedSettingsStore } from './advancedSettingsStore';

const advancedSettingsStore = useAdvancedSettingsStore(appPinia);
const { hasAdvancedSettingsAccess } = storeToRefs(advancedSettingsStore);

// 兼容层：旧代码仍从 advancedSettings.ts 读取可见性，新代码优先直接使用 store。
export { hasAdvancedSettingsAccess };

export function setAdvancedSettingsVisible(value: boolean): void {
  advancedSettingsStore.setAdvancedSettingsVisible(value);
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
