import {
  hasAdvancedSettingsAccess,
  normalizeAdvancedConfig,
  setAdvancedSettingsVisible,
} from './advancedSettings';
import { useAdvancedSettingsStore } from './advancedSettingsStore';
import { expect, it } from 'vitest';
import { appPinia } from '@/pinia';

it('高级设置字段转换与保存数据构造', async () => {
  expect(normalizeAdvancedConfig()).toEqual({
    show: false,
    enable: false,
    storyLogBackendUrl: '',
    storyLogApiVersion: '',
    storyLogBackendToken: '',
  });

  expect(
    normalizeAdvancedConfig({
      show: 1 as never,
      enable: 'yes' as never,
      storyLogBackendUrl: ' https://example.com ',
      storyLogApiVersion: ' v2 ',
      storyLogBackendToken: ' token ',
    })
  ).toEqual({
    show: true,
    enable: true,
    storyLogBackendUrl: 'https://example.com',
    storyLogApiVersion: 'v2',
    storyLogBackendToken: 'token',
  });

  const store = useAdvancedSettingsStore(appPinia);
  store.setAdvancedSettingsVisible(false);
  expect(hasAdvancedSettingsAccess.value).toEqual(false);

  setAdvancedSettingsVisible(true);
  expect(store.advancedSettingsVisible).toEqual(true);
  expect(hasAdvancedSettingsAccess.value).toEqual(true);
});
