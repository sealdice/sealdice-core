<template>
  <!-- 渲染-less 组件：只负责把 sw-update store 的状态翻译成全局提示 -->
  <span v-if="false" />
</template>

<script setup lang="ts">
// 渲染-less 组件：只负责把 sw-update store 的状态翻译成全局提示。
// 无未保存变更 → 提示条后自动刷新；有未保存变更 → 弹窗交用户决定。
import { useDialog, useMessage } from 'naive-ui';
import { storeToRefs } from 'pinia';
import { watch } from 'vue';
import { requestControlledReload } from '@/features/pwa/swUpdate';
import { useSwUpdateStore } from '@/features/pwa/swUpdateStore';
import { hasUnsavedChanges } from '@/features/unsavedChanges';
import { appPinia } from '@/pinia';

const AUTO_RELOAD_DELAY_MS = 1200;

const swUpdateStore = useSwUpdateStore(appPinia);
const { updateReady, updateReason, navigationTarget } = storeToRefs(swUpdateStore);
const dialog = useDialog();
const message = useMessage();

function applyUpdate() {
  if (swUpdateStore.applying) return;

  const reloaded = requestControlledReload(navigationTarget.value);
  if (!reloaded) {
    message.error('页面刷新过于频繁，请稍后手动刷新');
    return;
  }
  swUpdateStore.markApplying();
}

// immediate：升级状态可能早于组件挂载出现（如首屏 chunk 就 404），挂载时要补响应。
watch(
  updateReady,
  ready => {
    if (!ready) return;

    if (!hasUnsavedChanges.value) {
      message.loading('检测到新版本，正在刷新页面…', { duration: AUTO_RELOAD_DELAY_MS });
      window.setTimeout(applyUpdate, AUTO_RELOAD_DELAY_MS);
      return;
    }

    const isChunkError = updateReason.value === 'chunk-error';
    dialog.warning({
      title: isChunkError ? '页面资源已过期' : '发现新版本',
      content: isChunkError
        ? '当前页面版本过旧，部分功能无法正常加载，需要刷新页面。当前有未保存的更改，立即刷新会丢失这些更改。'
        : '控制台已更新到新版本，需要刷新页面加载最新内容。当前有未保存的更改，立即刷新会丢失这些更改。',
      positiveText: '立即刷新',
      negativeText: '稍后',
      onPositiveClick: applyUpdate,
      onNegativeClick: () => swUpdateStore.dismiss(),
      onClose: () => swUpdateStore.dismiss(),
      onMaskClick: () => swUpdateStore.dismiss(),
      onEsc: () => swUpdateStore.dismiss(),
    });
  },
  { immediate: true },
);
</script>
