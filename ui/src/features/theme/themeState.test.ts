import {
  createThemeStorage,
  isThemeMode,
  readStoredThemeMode,
  resolveThemeMode,
  syncDocumentTheme,
  writeStoredThemeMode,
} from './themeState';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertClass = (element: HTMLElement, className: string, expected: boolean) => {
  assertEqual(element.classList.contains(className), expected);
};

const storage = createThemeStorage();
assertEqual(readStoredThemeMode(storage), 'light');
writeStoredThemeMode(storage, 'dark');
assertEqual(readStoredThemeMode(storage), 'dark');
storage.setItem('sd-theme-mode', 'not-valid');
assertEqual(readStoredThemeMode(storage), 'light');

assertEqual(isThemeMode('system'), true);
assertEqual(isThemeMode('auto'), false);

assertEqual(resolveThemeMode('light', true), 'light');
assertEqual(resolveThemeMode('dark', false), 'dark');
assertEqual(resolveThemeMode('system', true), 'dark');
assertEqual(resolveThemeMode('system', false), 'light');

const classNames = new Set<string>();
const root = {
  classList: {
    contains(className: string) {
      return classNames.has(className);
    },
    toggle(className: string, force?: boolean) {
      const enabled = force ?? !classNames.has(className);
      if (enabled) {
        classNames.add(className);
      } else {
        classNames.delete(className);
      }
      return enabled;
    },
  },
  dataset: {},
  style: {},
} as HTMLElement;
syncDocumentTheme(root, 'dark');
assertClass(root, 'dark', true);
assertEqual(root.dataset.theme, 'dark');
assertEqual(root.style.colorScheme, 'dark');

syncDocumentTheme(root, 'light');
assertClass(root, 'dark', false);
assertEqual(root.dataset.theme, 'light');
assertEqual(root.style.colorScheme, 'light');
