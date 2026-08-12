import { storeToRefs } from 'pinia';
import { useThemeStore } from './store';

// 兼容包装器：现有调用方零改动。
// 新代码请直接用 const themeStore = useThemeStore()。
export function useAppTheme() {
  const store = useThemeStore();
  const { isDark, resolvedTheme, themeMode, themeOverrides } = storeToRefs(store);

  return {
    isDark,
    resolvedTheme,
    themeMode,
    themeOverrides,
    setThemeMode: store.setThemeMode,
    toggleTheme: store.toggleTheme,
  };
}
