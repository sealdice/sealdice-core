import { normalizeAdvancedConfig } from './advancedSettings';

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
  },
);
