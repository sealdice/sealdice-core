import {
  DEFAULT_THEME_PALETTE,
  THEME_PALETTE_STORAGE_KEY,
  createThemeOverrides,
  isThemeColorKey,
  isThemeHexColor,
  readStoredThemePalette,
  syncDocumentThemePalette,
  writeStoredThemePalette,
  type ThemePalette,
} from './themePalette';
import { createThemeStorage } from './themeState';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const storage = createThemeStorage();
assertDeepEqual(readStoredThemePalette(storage), DEFAULT_THEME_PALETTE);

storage.setItem(THEME_PALETTE_STORAGE_KEY, JSON.stringify({
  primary: '#0ea5e9',
  info: 'not-a-color',
  unknown: '#ffffff',
}));
assertDeepEqual(readStoredThemePalette(storage), {
  ...DEFAULT_THEME_PALETTE,
  primary: '#0ea5e9',
});

const customPalette: ThemePalette = {
  primary: '#0ea5e9',
  info: '#6366f1',
  success: '#22c55e',
  warning: '#f59e0b',
  error: '#ef4444',
};
writeStoredThemePalette(storage, customPalette);
assertDeepEqual(readStoredThemePalette(storage), customPalette);

assertEqual(isThemeColorKey('primary'), true);
assertEqual(isThemeColorKey('brand'), false);
assertEqual(isThemeHexColor('#ABCDEF'), true);
assertEqual(isThemeHexColor('#abcd'), false);

const lightOverrides = createThemeOverrides(customPalette, 'light');
assertEqual(lightOverrides.common?.primaryColor, '#0ea5e9');
assertEqual(lightOverrides.common?.infoColor, '#6366f1');
assertEqual(lightOverrides.Menu?.itemTextColor, '#ffffff');
assertEqual(lightOverrides.Menu?.itemIconColorCollapsed, '#ffffff');

const darkOverrides = createThemeOverrides(customPalette, 'dark');
assertEqual(darkOverrides.common?.bodyColor, '#0f172a');
assertEqual(darkOverrides.Drawer?.color, '#182133');

const styleValues = new Map<string, string>();
const fakeRoot = {
  style: {
    setProperty(name: string, value: string) {
      styleValues.set(name, value);
    },
  },
} as HTMLElement;

syncDocumentThemePalette(fakeRoot, customPalette);
assertEqual(styleValues.get('--sd-primary'), '#0ea5e9');
assertEqual(styleValues.get('--sd-error'), '#ef4444');
