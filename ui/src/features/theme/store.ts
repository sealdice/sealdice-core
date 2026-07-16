import { defineStore } from 'pinia';
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
import {
  DEFAULT_THEME_PALETTE,
  createThemeOverrides,
  normalizeThemePalette,
  readStoredThemePalette,
  syncDocumentThemePalette,
  writeStoredThemePalette,
  type ThemeColorKey,
  type ThemePalette,
} from './themePalette';

const storage = typeof window === 'undefined' ? undefined : window.localStorage;

export const useThemeStore = defineStore('theme', () => {
  const themeMode = ref<ThemeMode>(readStoredThemeMode(storage));
  const themePalette = ref<ThemePalette>(readStoredThemePalette(storage));
  const preferredDark = usePreferredDark();

  const resolvedTheme = computed<ResolvedTheme>(() =>
    resolveThemeMode(themeMode.value, preferredDark.value),
  );
  const isDark = computed(() => resolvedTheme.value === 'dark');
  const themeOverrides = computed(() =>
    createThemeOverrides(themePalette.value, resolvedTheme.value),
  );

  watch(
    resolvedTheme,
    theme => {
      if (typeof document === 'undefined') return;
      syncDocumentTheme(document.documentElement, theme);
      syncDocumentThemePalette(document.documentElement, themePalette.value);
    },
    { immediate: true },
  );

  watch(themeMode, mode => {
    writeStoredThemeMode(storage, mode);
  });

  watch(
    themePalette,
    palette => {
      writeStoredThemePalette(storage, palette);
      if (typeof document !== 'undefined') {
        syncDocumentThemePalette(document.documentElement, palette);
      }
    },
    { deep: true, immediate: true },
  );

  function setThemeMode(mode: ThemeMode) {
    themeMode.value = mode;
  }

  function toggleTheme() {
    themeMode.value = isDark.value ? 'light' : 'dark';
  }

  function setThemePalette(palette: ThemePalette) {
    themePalette.value = normalizeThemePalette(palette);
  }

  function setThemeColor(key: ThemeColorKey, color: string) {
    themePalette.value = normalizeThemePalette({
      ...themePalette.value,
      [key]: color,
    });
  }

  function resetThemePalette() {
    themePalette.value = { ...DEFAULT_THEME_PALETTE };
  }

  return {
    isDark,
    resolvedTheme,
    themeMode,
    themeOverrides,
    themePalette,
    setThemeMode,
    toggleTheme,
    setThemePalette,
    setThemeColor,
    resetThemePalette,
  };
});
