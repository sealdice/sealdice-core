<template>
  <n-tooltip placement="bottom">
    <template #trigger>
      <span class="theme-switch-trigger">
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
              <i-ep-moon v-if="isDark" />
              <i-ep-sunny v-else />
            </n-icon>
          </template>
        </n-button>
      </span>
    </template>
    {{ isDark ? '切换到亮色模式' : '切换到深色模式' }}
  </n-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useAppTheme } from '@/features/theme';

const { isDark, toggleTheme } = useAppTheme();
// Naive UI 的 quaternary 按钮颜色由 color prop 写入内部 token；亮色态用深色图标，深色态用黄色图标。
const switchIconColor = computed(() => (isDark.value ? 'var(--sd-accent)' : 'var(--sd-text-primary)'));

function toggle() {
  toggleTheme();
}
</script>

<style scoped>
.theme-switch-trigger {
  display: inline-flex;
}
</style>
