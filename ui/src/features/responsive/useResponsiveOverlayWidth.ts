import { computed } from 'vue';
import { useWindowSize } from '@vueuse/core';
import { resolveResponsiveOverlayWidth, type ResponsiveOverlayWidthOptions } from './layout';

export function useResponsiveOverlayWidth(options: ResponsiveOverlayWidthOptions) {
  const { width: viewportWidth } = useWindowSize();
  const width = computed(() => resolveResponsiveOverlayWidth(viewportWidth.value, options));

  return { width };
}
