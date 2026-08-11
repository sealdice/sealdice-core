import { computed, type MaybeRefOrGetter, type ShallowRef, toValue } from 'vue';
import { useElementSize } from '@vueuse/core';
import { resolveCompactContainerMode } from './layout';

export interface UseResponsiveContainerModeOptions {
  compactAt?: MaybeRefOrGetter<number>;
}

export function useResponsiveContainerMode(
  target: Readonly<ShallowRef<HTMLElement | null>>,
  options: UseResponsiveContainerModeOptions = {}
) {
  const { width } = useElementSize(target);
  const mode = computed(() =>
    resolveCompactContainerMode(width.value, toValue(options.compactAt ?? 760))
  );

  return { mode, width };
}
