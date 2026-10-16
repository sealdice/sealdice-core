import {
  DEFAULT_THEME_PALETTE,
  THEME_COLOR_KEYS,
  createThemeOverrides,
  syncDocumentThemePalette,
} from './themePalette';
import { expect, it } from 'vitest';

it('uses four fixed semantic colors and maps info to primary', async () => {
  expect(THEME_COLOR_KEYS.join(',')).toBe('primary,success,warning,error');
  expect(Object.keys(DEFAULT_THEME_PALETTE).join(',')).toBe('primary,success,warning,error');

  const lightOverrides = createThemeOverrides('light');
  expect(lightOverrides.common?.primaryColor).toBe(lightOverrides.common?.infoColor);
  expect(lightOverrides.common?.borderRadius).toBe('var(--sd-radius-sm)');
  expect(lightOverrides.common?.borderRadiusSmall).toBe('var(--sd-radius-xs)');
  expect(lightOverrides.Menu?.itemTextColor).toBe('var(--sd-text-inverse-soft)');
  expect(lightOverrides.Menu?.itemIconColorCollapsed).toBe('var(--sd-text-inverse)');
  expect(lightOverrides.Menu?.itemColorActive).toBe('var(--sd-bg-sidebar-selected)');
  expect(lightOverrides.Switch?.railColorActive).toBe('var(--sd-primary)');
  expect(lightOverrides.Switch?.loadingColor).toBe('var(--sd-primary)');
  expect(lightOverrides.Switch?.railWidthMedium).toBe('36px');
  expect(lightOverrides.Switch?.railHeightMedium).toBe('20px');

  const darkOverrides = createThemeOverrides('dark');
  expect(darkOverrides.common?.bodyColor).toBe('#0f172a');
  expect(darkOverrides.Drawer?.color).toBe('var(--sd-bg-elevated)');

  const styleValues = new Map<string, string>();
  const fakeRoot = {
    dataset: { theme: 'light' },
    style: {
      setProperty(name: string, value: string) {
        styleValues.set(name, value);
      },
    },
  } as unknown as HTMLElement;

  syncDocumentThemePalette(fakeRoot);
  expect(styleValues.get('--sd-primary')).toBe('#2563eb');
  expect(styleValues.get('--sd-info')).toBe(styleValues.get('--sd-primary'));
  expect(styleValues.get('--sd-accent')).toBe(styleValues.get('--sd-primary'));
  expect(styleValues.has('--sd-success')).toBe(true);
  expect(styleValues.has('--sd-warning')).toBe(true);
  expect(styleValues.has('--sd-error')).toBe(true);
});
