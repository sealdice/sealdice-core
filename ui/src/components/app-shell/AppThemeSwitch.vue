<template>
  <n-tooltip placement="bottom">
    <template #trigger>
      <span ref="triggerRef" class="theme-switch-trigger">
        <n-button
          quaternary
          circle
          class="theme-switch"
          :color="switchIconColor"
          :aria-label="isDark ? '切换到亮色模式' : '切换到深色模式'"
          @click="toggle"
        >
          <template #icon>
            <n-icon size="1.25rem">
              <i-carbon-moon v-if="isDark" />
              <i-carbon-sun v-else />
            </n-icon>
          </template>
        </n-button>
      </span>
    </template>
    {{ isDark ? '切换到亮色模式' : '切换到深色模式' }}
  </n-tooltip>
</template>

<script setup lang="ts">
import { computed, inject, ref } from 'vue';
import { useAppTheme } from '@/features/theme';
import { triggerThemeTransitionKey } from '@/features/theme/themeTransition';

const { isDark } = useAppTheme();
const triggerThemeTransition = inject(triggerThemeTransitionKey);
const triggerRef = ref<HTMLElement | null>(null);
// Naive UI 的 quaternary 按钮颜色由 color prop 写入内部 token；亮色态用深色图标，深色态用黄色图标。
const switchIconColor = computed(() => (isDark.value ? 'var(--sd-accent)' : 'var(--sd-text-primary)'));

function toggle(event: MouseEvent) {
  const rect = triggerRef.value?.getBoundingClientRect();
  triggerThemeTransition?.(rect ?? event);
}
</script>

<style scoped>
.theme-switch-trigger {
  display: inline-flex;
}
</style>
