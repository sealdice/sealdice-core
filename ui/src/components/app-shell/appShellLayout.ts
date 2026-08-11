import type { AppLayoutName } from '@/router/types';

export type AppShellContentMode = 'default' | 'wide';
export type AppShellContainerMode = 'default' | 'workspace';
export type AppShellViewportMode = 'mobile' | 'tablet' | 'desktop';

export const APP_SHELL_MOBILE_MAX_WIDTH = 767.9;
export const APP_SHELL_TABLET_MIN_WIDTH = 768;
export const APP_SHELL_DESKTOP_MIN_WIDTH = 1024;

// AppShell 的布局差异只体现在内容区宽度，不改变侧边栏、面包屑和解锁弹窗。
// default 适合阅读型后台页，wide 适合编辑器、日志、diff、复杂表格。
export function getAppShellContentClass(mode: AppShellContentMode): string {
  return mode === 'wide' ? 'sd-main-container sd-main-container--wide' : 'sd-main-container';
}

export function getAppShellContentMode(layout?: AppLayoutName): AppShellContentMode {
  return layout === 'wide' || layout === 'workspace' ? 'wide' : 'default';
}

export function getAppShellContainerMode(layout?: AppLayoutName): AppShellContainerMode {
  return layout === 'workspace' ? 'workspace' : 'default';
}

export function getAppShellContainerClass(mode: AppShellContainerMode): string {
  return mode === 'workspace' ? 'sd-page-shell sd-page-shell--workspace' : 'sd-page-shell';
}

export function getAppShellDrawerWidth(): string {
  return 'min(200px, 76vw)';
}

export function isAppShellMobileWidth(width: number): boolean {
  return width <= APP_SHELL_MOBILE_MAX_WIDTH;
}

export function getAppShellViewportMode(width: number): AppShellViewportMode {
  if (isAppShellMobileWidth(width)) return 'mobile';
  if (width < APP_SHELL_DESKTOP_MIN_WIDTH) return 'tablet';
  return 'desktop';
}

export function shouldCollapseAppShellSidebar(mode: AppShellViewportMode): boolean {
  return mode !== 'desktop';
}
