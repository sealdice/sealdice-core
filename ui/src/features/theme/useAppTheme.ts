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
const themeMode = ref<ThemeMode>(readStoredThemeMode(storage));
const themePalette = ref<ThemePalette>(readStoredThemePalette(storage));
const preferredDark = usePreferredDark();

const resolvedTheme = computed<ResolvedTheme>(() =>
  resolveThemeMode(themeMode.value, preferredDark.value),
);
const isDark = computed(() => resolvedTheme.value === 'dark');
// Naive UI 的 token 由 mode + palette 推导，App.vue 只消费最终结果，避免根组件继续累积主题细节。
const themeOverrides = computed(() => createThemeOverrides(themePalette.value, resolvedTheme.value));

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

watch(themePalette, palette => {
  writeStoredThemePalette(storage, palette);
  if (typeof document !== 'undefined') {
    syncDocumentThemePalette(document.documentElement, palette);
  }
}, { deep: true, immediate: true });

export function useAppTheme() {
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
    resetThemePalette,
    setThemeColor,
    setThemeMode,
    setThemePalette,
    themeMode,
    themeOverrides,
    themePalette,
    toggleTheme,
  };
}
