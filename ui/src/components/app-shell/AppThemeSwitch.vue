<template>
  <n-tooltip placement="bottom">
    <template #trigger>
      <span class="theme-switch-trigger">
        <n-button
          quaternary
          circle
          class="theme-switch"
          :color="switchIconColor"
          :aria-label="tooltipText"
          @click="toggle"
        >
          <template #icon>
            <n-icon size="1.25rem">
              <i-tabler-sun v-if="themeMode === 'light'" />
              <i-tabler-moon v-else-if="themeMode === 'dark'" />
              <i-tabler-sun-moon v-else />
            </n-icon>
          </template>
        </n-button>
      </span>
    </template>
    {{ tooltipText }}
  </n-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useAppTheme } from '@/features/theme';

const { isDark, themeMode, toggleTheme } = useAppTheme();
const switchIconColor = computed(() =>
  themeMode.value === 'dark' ? 'var(--sd-accent)' : 'var(--sd-text-primary)'
);
const tooltipText = computed(() => {
  if (themeMode.value === 'light') return '亮色模式・点击切换到深色模式';
  if (themeMode.value === 'dark') return '深色模式・点击切换为跟随系统';
  return `跟随系统（当前${isDark.value ? '深色' : '亮色'}）・点击切换到亮色模式`;
});

function toggle() {
  toggleTheme();
}
</script>

<style scoped>
.theme-switch-trigger {
  display: inline-flex;
}
</style>
