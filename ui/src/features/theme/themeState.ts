export type ThemeMode = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'sd-theme-mode';
export const DEFAULT_THEME_MODE: ThemeMode = 'light';

export interface ThemeStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export function isThemeMode(value: unknown): value is ThemeMode {
  return value === 'light' || value === 'dark' || value === 'system';
}

export function createThemeStorage(): ThemeStorage {
  const values = new Map<string, string>();

  return {
    getItem(key: string) {
      return values.get(key) ?? null;
    },
    setItem(key: string, value: string) {
      values.set(key, value);
    },
  };
}

export function readStoredThemeMode(storage: ThemeStorage | undefined): ThemeMode {
  if (!storage) return DEFAULT_THEME_MODE;

  try {
    const value = storage.getItem(THEME_STORAGE_KEY);
    return isThemeMode(value) ? value : DEFAULT_THEME_MODE;
  } catch {
    return DEFAULT_THEME_MODE;
  }
}

export function writeStoredThemeMode(storage: ThemeStorage | undefined, mode: ThemeMode): void {
  if (!storage) return;

  try {
    storage.setItem(THEME_STORAGE_KEY, mode);
  } catch {
    // localStorage can be unavailable in private or embedded contexts.
  }
}

export function resolveThemeMode(mode: ThemeMode, systemPrefersDark: boolean): ResolvedTheme {
  if (mode === 'system') return systemPrefersDark ? 'dark' : 'light';
  return mode;
}

export function syncDocumentTheme(root: HTMLElement | undefined, theme: ResolvedTheme): void {
  if (!root) return;

  root.classList.toggle('dark', theme === 'dark');
  root.dataset.theme = theme;
  root.style.colorScheme = theme;
}
