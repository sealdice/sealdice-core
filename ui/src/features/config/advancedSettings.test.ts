import { normalizeAdvancedConfig } from './advancedSettings';
import { expect, it } from 'vitest';

it('高级设置字段转换与保存数据构造', () => {
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
});
