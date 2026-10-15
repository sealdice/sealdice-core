import type { AdvancedConfig } from '@/api';

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
