import {
  DEFAULT_THEME_PALETTE,
  THEME_COLOR_KEYS,
  createThemeOverrides,
  syncDocumentThemePalette,
} from './themePalette';
import { it } from 'vitest';

it('uses four fixed semantic colors and maps info to primary', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  assertEqual(THEME_COLOR_KEYS.join(','), 'primary,success,warning,error');
  assertEqual(Object.keys(DEFAULT_THEME_PALETTE).join(','), 'primary,success,warning,error');

  const lightOverrides = createThemeOverrides('light');
  assertEqual(lightOverrides.common?.primaryColor, lightOverrides.common?.infoColor);
  assertEqual(lightOverrides.common?.borderRadius, 'var(--sd-radius-sm)');
  assertEqual(lightOverrides.common?.borderRadiusSmall, 'var(--sd-radius-xs)');
  assertEqual(lightOverrides.Menu?.itemTextColor, 'var(--sd-text-inverse-soft)');
  assertEqual(lightOverrides.Menu?.itemIconColorCollapsed, 'var(--sd-text-inverse)');
  assertEqual(lightOverrides.Menu?.itemColorActive, 'var(--sd-bg-sidebar-selected)');

  const darkOverrides = createThemeOverrides('dark');
  assertEqual(darkOverrides.common?.bodyColor, '#0f172a');
  assertEqual(darkOverrides.Drawer?.color, 'var(--sd-bg-elevated)');

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
  assertEqual(styleValues.get('--sd-primary'), '#2563eb');
  assertEqual(styleValues.get('--sd-info'), styleValues.get('--sd-primary'));
  assertEqual(styleValues.get('--sd-accent'), styleValues.get('--sd-primary'));
  assertEqual(styleValues.has('--sd-success'), true);
  assertEqual(styleValues.has('--sd-warning'), true);
  assertEqual(styleValues.has('--sd-error'), true);
});
