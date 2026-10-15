import {
  createThemeStorage,
  isThemeMode,
  nextThemeMode,
  readStoredThemeMode,
  resolveThemeMode,
  shouldTransitionTheme,
  syncDocumentTheme,
  writeStoredThemeMode,
} from './themeState';
import { expect, it } from 'vitest';

it('主题偏好解析、切换和持久化', async () => {
  const assertClass = (element: HTMLElement, className: string, expected: boolean) => {
    expect(element.classList.contains(className)).toBe(expected);
  };

  const storage = createThemeStorage();
  expect(readStoredThemeMode(storage)).toBe('system');
  writeStoredThemeMode(storage, 'dark');
  expect(readStoredThemeMode(storage)).toBe('dark');
  storage.setItem('sd-theme-mode', 'not-valid');
  expect(readStoredThemeMode(storage)).toBe('system');

  expect(isThemeMode('system')).toBe(true);
  expect(isThemeMode('auto')).toBe(false);

  expect(resolveThemeMode('light', true)).toBe('light');
  expect(resolveThemeMode('dark', false)).toBe('dark');
  expect(resolveThemeMode('system', true)).toBe('dark');
  expect(resolveThemeMode('system', false)).toBe('light');
  expect(nextThemeMode('light')).toBe('dark');
  expect(nextThemeMode('dark')).toBe('system');
  expect(nextThemeMode('system')).toBe('light');
  expect(shouldTransitionTheme('dark', 'system', true)).toBe(false);
  expect(shouldTransitionTheme('dark', 'system', false)).toBe(true);
  expect(shouldTransitionTheme('light', 'system', false)).toBe(false);
  expect(shouldTransitionTheme('light', 'system', true)).toBe(true);

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
  expect(root.dataset.theme).toBe('dark');
  expect(root.style.colorScheme).toBe('dark');

  syncDocumentTheme(root, 'light');
  assertClass(root, 'dark', false);
  expect(root.dataset.theme).toBe('light');
  expect(root.style.colorScheme).toBe('light');
});
