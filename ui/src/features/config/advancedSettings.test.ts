import {
  hasAdvancedSettingsAccess,
  normalizeAdvancedConfig,
  setAdvancedSettingsVisible,
} from './advancedSettings';
import { useAdvancedSettingsStore } from './advancedSettingsStore';
import { it } from 'vitest';
import { appPinia } from '@/pinia';

it('passes', async () => {
  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    const actualText = JSON.stringify(actual);
    const expectedText = JSON.stringify(expected);
    if (actualText !== expectedText) {
      throw new Error(`expected ${expectedText}, got ${actualText}`);
    }
  };

  assertDeepEqual(normalizeAdvancedConfig(), {
    show: false,
    enable: false,
    storyLogBackendUrl: '',
    storyLogApiVersion: '',
    storyLogBackendToken: '',
  });

  assertDeepEqual(
    normalizeAdvancedConfig({
      show: 1 as never,
      enable: 'yes' as never,
      storyLogBackendUrl: ' https://example.com ',
      storyLogApiVersion: ' v2 ',
      storyLogBackendToken: ' token ',
    }),
    {
      show: true,
      enable: true,
      storyLogBackendUrl: 'https://example.com',
      storyLogApiVersion: 'v2',
      storyLogBackendToken: 'token',
    }
  );

  const store = useAdvancedSettingsStore(appPinia);
  store.setAdvancedSettingsVisible(false);
  assertDeepEqual(hasAdvancedSettingsAccess.value, false);

  setAdvancedSettingsVisible(true);
  assertDeepEqual(store.advancedSettingsVisible, true);
  assertDeepEqual(hasAdvancedSettingsAccess.value, true);
});
