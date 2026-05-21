export type AppShellContentMode = 'default' | 'wide';

// AppShell 的布局差异只体现在内容区宽度，不改变侧边栏、面包屑和解锁弹窗。
// default 适合阅读型后台页，wide 适合编辑器、日志、diff、复杂表格。
export function getAppShellContentClass(mode: AppShellContentMode): string {
  return mode === 'wide'
    ? 'sd-main-container sd-main-container--wide'
    : 'sd-main-container';
}

export function getAppShellDrawerWidth(): string {
  return 'min(320px, 86vw)';
}
