export type AppShellContentMode = 'default' | 'wide';

export function getAppShellContentClass(mode: AppShellContentMode): string {
  return mode === 'wide'
    ? 'sd-main-container sd-main-container--wide'
    : 'sd-main-container';
}
