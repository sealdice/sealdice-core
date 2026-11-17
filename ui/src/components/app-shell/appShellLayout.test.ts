import {
  getAppShellContainerClass,
  getAppShellContentClass,
  getAppShellDrawerWidth,
  getAppShellViewportMode,
  shouldCollapseAppShellSidebar,
} from './appShellLayout.ts';
import { describe, expect, it } from 'vitest';

describe('app shell layout', () => {
  it('keeps the established content and container classes', () => {
    expect(getAppShellContentClass('default')).toBe('sd-main-container');
    expect(getAppShellContentClass('wide')).toBe('sd-main-container sd-main-container--wide');
    expect(getAppShellContainerClass('default')).toBe('sd-page-shell');
    expect(getAppShellContainerClass('workspace')).toBe('sd-page-shell sd-page-shell--workspace');
    expect(getAppShellDrawerWidth()).toBe('min(200px, 76vw)');
  });

  it('uses mobile, tablet and desktop modes at the exact boundaries', () => {
    expect(getAppShellViewportMode(320)).toBe('mobile');
    expect(getAppShellViewportMode(767)).toBe('mobile');
    expect(getAppShellViewportMode(768)).toBe('tablet');
    expect(getAppShellViewportMode(1023.9)).toBe('tablet');
    expect(getAppShellViewportMode(1024)).toBe('desktop');
  });

  it('defaults tablet sidebars to collapsed and desktop sidebars to expanded', () => {
    expect(shouldCollapseAppShellSidebar('mobile')).toBe(true);
    expect(shouldCollapseAppShellSidebar('tablet')).toBe(true);
    expect(shouldCollapseAppShellSidebar('desktop')).toBe(false);
  });
});
