import { computed, ref, watch } from 'vue';
import { usePreferredDark } from '@vueuse/core';
import {
  type ResolvedTheme,
  type ThemeMode,
  readStoredThemeMode,
  resolveThemeMode,
  syncDocumentTheme,
  writeStoredThemeMode,
} from './themeState';

const storage = typeof window === 'undefined' ? undefined : window.localStorage;
const themeMode = ref<ThemeMode>(readStoredThemeMode(storage));
const preferredDark = usePreferredDark();

const resolvedTheme = computed<ResolvedTheme>(() =>
  resolveThemeMode(themeMode.value, preferredDark.value),
);
const isDark = computed(() => resolvedTheme.value === 'dark');

watch(
  resolvedTheme,
  theme => {
    if (typeof document === 'undefined') return;
    syncDocumentTheme(document.documentElement, theme);
  },
  { immediate: true },
);

watch(themeMode, mode => {
  writeStoredThemeMode(storage, mode);
});

export function useAppTheme() {
  function setThemeMode(mode: ThemeMode) {
    themeMode.value = mode;
  }

  function toggleTheme() {
    themeMode.value = isDark.value ? 'light' : 'dark';
  }

  return {
    isDark,
    resolvedTheme,
    setThemeMode,
    themeMode,
    toggleTheme,
  };
}
