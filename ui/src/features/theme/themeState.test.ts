// @vitest-environment happy-dom
import {
  nextThemeMode,
  readStoredThemeMode,
  resolveThemeMode,
  shouldTransitionTheme,
  syncDocumentTheme,
  writeStoredThemeMode,
} from './themeState';
import { expect, it } from 'vitest';

it('主题偏好保存后恢复，非法偏好恢复为跟随系统', () => {
  localStorage.clear();
  expect(readStoredThemeMode(localStorage)).toBe('system');
  writeStoredThemeMode(localStorage, 'dark');
  expect(readStoredThemeMode(localStorage)).toBe('dark');
  localStorage.setItem('sd-theme-mode', 'not-valid');
  expect(readStoredThemeMode(localStorage)).toBe('system');
  localStorage.clear();
});

it('主题循环切换，并且仅在实际颜色变化时播放过渡', () => {
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
});

it('文档颜色标识随主题同步', () => {
  const root = document.createElement('div');
  syncDocumentTheme(root, 'dark');
  expect(root.classList.contains('dark')).toBe(true);
  expect(root.dataset.theme).toBe('dark');
  expect(root.style.colorScheme).toBe('dark');

  syncDocumentTheme(root, 'light');
  expect(root.classList.contains('dark')).toBe(false);
  expect(root.dataset.theme).toBe('light');
  expect(root.style.colorScheme).toBe('light');
});
