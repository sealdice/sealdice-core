<template>
  <n-tooltip placement="bottom">
    <template #trigger>
      <n-button
        quaternary
        circle
        class="theme-palette-button"
        aria-label="打开主题设置"
        @click="dialogVisible = true"
      >
        <template #icon>
          <n-icon size="1.2rem">
            <i-carbon-paint-brush />
          </n-icon>
        </template>
      </n-button>
    </template>
    主题设置
  </n-tooltip>

  <AppThemePaletteDialog v-model:show="dialogVisible" />
</template>

<script setup lang="ts">
import { ref } from 'vue';
import AppThemePaletteDialog from './AppThemePaletteDialog.vue';

const dialogVisible = ref(false);
</script>

<style scoped>
.theme-palette-button {
  color: var(--sd-accent-strong);
}
</style>
