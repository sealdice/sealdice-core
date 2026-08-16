import {
  DEFAULT_THEME_PALETTE,
  THEME_COLOR_KEYS,
  createThemeOverrides,
  syncDocumentThemePalette,
} from './themePalette';
import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

const baseCss = readFileSync(new URL('../../assets/base.css', import.meta.url), 'utf8');

function readThemeVariables(selector: ':root' | '.dark'): Record<string, string> {
  const block = baseCss.match(new RegExp(`\\${selector}\\s*\\{([\\s\\S]*?)\\n\\}`))?.[1];
  if (!block) throw new Error(`missing ${selector} theme variables`);

  return Object.fromEntries(
    [...block.matchAll(/--([\w-]+):\s*([^;]+);/g)].map(([, name, value]) => [name, value.trim()])
  );
}

it('uses four fixed semantic colors and maps info to primary', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  assertEqual(THEME_COLOR_KEYS.join(','), 'primary,success,warning,error');
  assertEqual(Object.keys(DEFAULT_THEME_PALETTE).join(','), 'primary,success,warning,error');

  const lightOverrides = createThemeOverrides('light');
  assertEqual(lightOverrides.common?.primaryColor, '#3069e5');
  assertEqual(lightOverrides.common?.primaryColor, lightOverrides.common?.infoColor);
  assertEqual(lightOverrides.common?.successColor, '#218755');
  assertEqual(lightOverrides.common?.warningColor, '#a26b0c');
  assertEqual(lightOverrides.common?.errorColor, '#dd2929');
  assertEqual(lightOverrides.common?.borderRadius, 'var(--sd-radius-sm)');
  assertEqual(lightOverrides.common?.borderRadiusSmall, 'var(--sd-radius-xs)');
  assertEqual(lightOverrides.Menu?.itemTextColor, 'var(--sd-text-inverse-soft)');
  assertEqual(lightOverrides.Menu?.itemIconColorCollapsed, 'var(--sd-text-inverse)');
  assertEqual(lightOverrides.Menu?.itemColorActive, 'var(--sd-bg-sidebar-selected)');
  assertEqual(lightOverrides.Switch?.railColorActive, 'var(--sd-primary)');
  assertEqual(lightOverrides.Switch?.loadingColor, 'var(--sd-primary)');
  assertEqual(lightOverrides.Switch?.railWidthMedium, '36px');
  assertEqual(lightOverrides.Switch?.railHeightMedium, '20px');

  const darkOverrides = createThemeOverrides('dark');
  assertEqual(darkOverrides.common?.bodyColor, '#0f172a');
  assertEqual(darkOverrides.common?.primaryColor, '#6791f1');
  assertEqual(darkOverrides.common?.successColor, '#4cac73');
  assertEqual(darkOverrides.common?.warningColor, '#ca9438');
  assertEqual(darkOverrides.common?.errorColor, '#ea6666');
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
  assertEqual(styleValues.get('--sd-primary'), '#3069e5');
  assertEqual(styleValues.get('--sd-success'), '#218755');
  assertEqual(styleValues.get('--sd-warning'), '#a26b0c');
  assertEqual(styleValues.get('--sd-error'), '#dd2929');
  assertEqual(styleValues.get('--sd-info'), styleValues.get('--sd-primary'));
  assertEqual(styleValues.get('--sd-accent'), styleValues.get('--sd-primary'));
  assertEqual(styleValues.get('--sd-accent-strong'), '#1f4cbf');
  assertEqual(styleValues.has('--sd-success'), true);
  assertEqual(styleValues.has('--sd-warning'), true);
  assertEqual(styleValues.has('--sd-error'), true);

  fakeRoot.dataset.theme = 'dark';
  syncDocumentThemePalette(fakeRoot);
  assertEqual(styleValues.get('--sd-primary'), '#6791f1');
  assertEqual(styleValues.get('--sd-success'), '#4cac73');
  assertEqual(styleValues.get('--sd-warning'), '#ca9438');
  assertEqual(styleValues.get('--sd-error'), '#ea6666');
  assertEqual(styleValues.get('--sd-accent-strong'), '#89a9ea');
});

it('keeps initial CSS colors aligned with runtime theme colors', () => {
  const lightVariables = readThemeVariables(':root');
  const darkVariables = readThemeVariables('.dark');
  const lightOverrides = createThemeOverrides('light');
  const darkOverrides = createThemeOverrides('dark');

  for (const key of THEME_COLOR_KEYS) {
    expect(lightVariables[`sd-${key}`]).toBe(lightOverrides.common?.[`${key}Color`]);
    expect(darkVariables[`sd-${key}`]).toBe(darkOverrides.common?.[`${key}Color`]);
  }

  expect(lightVariables['sd-accent-strong']).toBe('#1f4cbf');
  expect(darkVariables['sd-accent-strong']).toBe('#89a9ea');
  expect(lightVariables['sd-bg-sidebar']).toBe('#243247');
  expect(darkVariables['sd-bg-sidebar']).toBe('#090e18');
});
