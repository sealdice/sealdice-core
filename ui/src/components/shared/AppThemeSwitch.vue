<script setup lang="ts">
import { inject, ref } from 'vue';
import { useAppTheme } from '@/features/theme';
import { triggerThemeTransitionKey } from '@/features/theme/themeTransition';

const { isDark } = useAppTheme();
const triggerThemeTransition = inject(triggerThemeTransitionKey);
const triggerRef = ref<HTMLElement | null>(null);

function toggle(event: MouseEvent) {
  const rect = triggerRef.value?.getBoundingClientRect();
  triggerThemeTransition?.(rect ?? event);
}
</script>

<template>
  <n-tooltip placement="bottom">
    <template #trigger>
      <span ref="triggerRef" class="theme-switch-trigger">
        <n-button
          quaternary
          circle
          class="theme-switch"
          :class="{ active: isDark }"
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

<style scoped>
.theme-switch-trigger {
  display: inline-flex;
}

.theme-switch {
  color: var(--sd-text-inverse);
}

.theme-switch.active {
  color: var(--sd-accent);
}
</style>
