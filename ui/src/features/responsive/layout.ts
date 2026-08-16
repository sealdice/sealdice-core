export type ResponsiveDataViewMode = 'compact' | 'table';

export interface ResponsiveOverlayWidthOptions {
  maxWidth: number;
  gutter?: number;
}

export function resolveCompactContainerMode(
  containerWidth: number,
  compactAt: number
): ResponsiveDataViewMode {
  return containerWidth <= compactAt ? 'compact' : 'table';
}

export function resolveResponsiveOverlayWidth(
  viewportWidth: number,
  options: ResponsiveOverlayWidthOptions
): number {
  const gutter = options.gutter ?? 16;
  return Math.max(0, Math.min(options.maxWidth, viewportWidth - gutter * 2));
}
