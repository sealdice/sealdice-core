import { defineStore } from 'pinia';
import { computed, nextTick, ref, watch } from 'vue';
import { usePreferredDark } from '@vueuse/core';
import {
  type ResolvedTheme,
  type ThemeMode,
  nextThemeMode,
  readStoredThemeMode,
  resolveThemeMode,
  shouldTransitionTheme,
  syncDocumentTheme,
  writeStoredThemeMode,
} from './themeState';
import { createThemeOverrides, syncDocumentThemePalette } from './themePalette';

const storage = typeof window === 'undefined' ? undefined : window.localStorage;
const THEME_TRANSITION_DISABLED_CLASS = 'sd-theme-transition-disabled';
const THEME_TRANSITIONING_CLASS = 'sd-theme-transitioning';
const THEME_VIEW_TRANSITIONING_CLASS = 'sd-theme-view-transitioning';
const THEME_TRANSITION_DURATION = 500;
let themeTransitionResetFrame: number | undefined;
let themeTransitionResetTimer: number | undefined;
let isNativeThemeTransitionActive = false;

function suppressThemeTransitions(): void {
  if (typeof document === 'undefined') return;

  const root = document.documentElement;
  root.classList.add(THEME_TRANSITION_DISABLED_CLASS);

  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
    root.classList.remove(THEME_TRANSITION_DISABLED_CLASS);
    return;
  }

  if (themeTransitionResetFrame !== undefined) {
    window.cancelAnimationFrame(themeTransitionResetFrame);
  }

  themeTransitionResetFrame = window.requestAnimationFrame(() => {
    themeTransitionResetFrame = window.requestAnimationFrame(() => {
      root.classList.remove(THEME_TRANSITION_DISABLED_CLASS);
      themeTransitionResetFrame = undefined;
    });
  });
}

function animateThemeTransition(): void {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return;

  const root = document.documentElement;

  if (themeTransitionResetTimer !== undefined) {
    window.clearTimeout(themeTransitionResetTimer);
  }

  root.classList.remove(THEME_TRANSITIONING_CLASS);
  // 读取布局以便连续快速切换时也能重新触发渐变动画。
  void root.offsetWidth;
  root.classList.add(THEME_TRANSITIONING_CLASS);

  themeTransitionResetTimer = window.setTimeout(() => {
    root.classList.remove(THEME_TRANSITIONING_CLASS);
    themeTransitionResetTimer = undefined;
  }, THEME_TRANSITION_DURATION);
}

export const useThemeStore = defineStore('theme', () => {
  const themeMode = ref<ThemeMode>(readStoredThemeMode(storage));
  const preferredDark = usePreferredDark();

  const resolvedTheme = computed<ResolvedTheme>(() =>
    resolveThemeMode(themeMode.value, preferredDark.value)
  );
  const isDark = computed(() => resolvedTheme.value === 'dark');
  const themeOverrides = computed(() => createThemeOverrides(resolvedTheme.value));
  let hasSyncedInitialTheme = false;

  watch(
    resolvedTheme,
    theme => {
      if (typeof document === 'undefined') return;
      if (hasSyncedInitialTheme && !isNativeThemeTransitionActive) {
        animateThemeTransition();
      } else {
        suppressThemeTransitions();
        hasSyncedInitialTheme = true;
      }
      syncDocumentTheme(document.documentElement, theme);
      syncDocumentThemePalette(document.documentElement);
      const themeColorMeta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
      themeColorMeta?.setAttribute('content', theme === 'dark' ? '#0f172a' : '#f4f6f9');
    },
    { immediate: true }
  );

  watch(themeMode, mode => {
    writeStoredThemeMode(storage, mode);
  });

  function setThemeMode(mode: ThemeMode) {
    if (themeMode.value === mode) return;

    if (!shouldTransitionTheme(themeMode.value, mode, preferredDark.value)) {
      themeMode.value = mode;
      return;
    }

    if (
      typeof document === 'undefined' ||
      typeof window === 'undefined' ||
      !document.startViewTransition ||
      window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
    ) {
      themeMode.value = mode;
      return;
    }

    const root = document.documentElement;
    isNativeThemeTransitionActive = true;
    root.classList.add(THEME_VIEW_TRANSITIONING_CLASS);

    const transition = document.startViewTransition(async () => {
      themeMode.value = mode;
      await nextTick();
    });

    void transition.finished.finally(() => {
      root.classList.remove(THEME_VIEW_TRANSITIONING_CLASS);
      isNativeThemeTransitionActive = false;
    });
  }

  function toggleTheme() {
    setThemeMode(nextThemeMode(themeMode.value));
  }

  return {
    isDark,
    resolvedTheme,
    themeMode,
    themeOverrides,
    setThemeMode,
    toggleTheme,
  };
});
