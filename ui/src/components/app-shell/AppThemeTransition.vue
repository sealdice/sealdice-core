<template>
  <div
    aria-hidden="true"
    class="theme-transition-overlay"
    :class="{ active, dark: isDark }"
    :style="overlayStyle"
    @transitionend="handleTransitionEnd"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { usePreferredReducedMotion } from '@vueuse/core';
import { useAppTheme } from '@/features/theme';

const { isDark, toggleTheme } = useAppTheme();
const prefersReducedMotion = usePreferredReducedMotion();
const active = ref(false);
const origin = ref({ x: 0, y: 0 });

const overlayStyle = computed(() => ({
  left: `${origin.value.x}px`,
  top: `${origin.value.y}px`,
}));

function resolveOrigin(source?: DOMRect | MouseEvent) {
  if (source instanceof MouseEvent) {
    return {
      x: source.clientX,
      y: source.clientY,
    };
  }

  if (source) {
    return {
      x: source.left + source.width / 2,
      y: source.top + source.height / 2,
    };
  }

  return {
    x: window.innerWidth - 32,
    y: 32,
  };
}

function toggle(source?: DOMRect | MouseEvent) {
  origin.value = resolveOrigin(source);

  if (prefersReducedMotion.value === 'reduce') {
    toggleTheme();
    return;
  }

  active.value = false;
  requestAnimationFrame(() => {
    active.value = true;
    toggleTheme();
  });
}

function handleTransitionEnd(event: TransitionEvent) {
  if (event.propertyName !== 'transform') return;
  active.value = false;
}

defineExpose({
  toggle,
});
</script>

<style scoped>
.theme-transition-overlay {
  position: fixed;
  z-index: 2147483647;
  width: 1px;
  height: 1px;
  border-radius: 9999px;
  background: var(--sd-bg-app);
  pointer-events: none;
  transform: translate(-50%, -50%) scale(0);
  transition: transform 620ms ease-in-out;
}

.theme-transition-overlay.active {
  transform: translate(-50%, -50%) scale(3000);
}

@media (prefers-reduced-motion: reduce) {
  .theme-transition-overlay {
    display: none;
    transition: none;
  }
}
</style>
