import { describe, expect, it } from 'vitest';
import { resolveCompactContainerMode, resolveResponsiveOverlayWidth } from './layout';

describe('resolveCompactContainerMode', () => {
  it('uses the compact view at and below the configured threshold', () => {
    expect(resolveCompactContainerMode(760, 760)).toBe('compact');
    expect(resolveCompactContainerMode(759.9, 760)).toBe('compact');
  });

  it('uses the table view above the configured threshold', () => {
    expect(resolveCompactContainerMode(760.1, 760)).toBe('table');
  });
});

describe('resolveResponsiveOverlayWidth', () => {
  it('keeps the requested maximum width when the viewport has room', () => {
    expect(resolveResponsiveOverlayWidth(1440, { maxWidth: 720, gutter: 16 })).toBe(720);
  });

  it('leaves equal safety gutters on narrow viewports', () => {
    expect(resolveResponsiveOverlayWidth(390, { maxWidth: 720, gutter: 16 })).toBe(358);
    expect(resolveResponsiveOverlayWidth(320, { maxWidth: 420, gutter: 16 })).toBe(288);
  });

  it('never returns a negative width', () => {
    expect(resolveResponsiveOverlayWidth(20, { maxWidth: 420, gutter: 16 })).toBe(0);
  });
});
