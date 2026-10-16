<template>
  <n-alert
    v-if="visible"
    class="sd-beta-notice"
    type="info"
    :show-icon="true"
    closable
    @close="handleClose"
  >
    <div class="sd-beta-notice__body">
      <span class="sd-beta-notice__text">
        新版 UI 处于公测阶段，功能与稳定性仍在打磨中。遇到问题欢迎
        <a
          class="sd-beta-notice__link"
          :href="feedbackUrl"
          target="_blank"
          rel="noopener noreferrer"
          >反馈</a
        >；如果暂时无法正常使用，可以切回旧版 UI。
      </span>
      <n-button size="small" secondary tag="a" :href="oldUIUrl">切换到旧版 UI</n-button>
    </div>
  </n-alert>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { resolveSwitchToOldUIUrl } from '@/api/config';
import {
  UI_BETA_FEEDBACK_URL,
  persistUiBetaNoticeDismissed,
  readUiBetaNoticeDismissed,
} from '@/features/uiBetaNotice/notice';

const feedbackUrl = UI_BETA_FEEDBACK_URL;
const oldUIUrl =
  typeof window !== 'undefined' ? resolveSwitchToOldUIUrl(window.location) : '/?ui=old';
const visible = ref(
  !(typeof localStorage !== 'undefined' && readUiBetaNoticeDismissed(localStorage))
);

function handleClose() {
  visible.value = false;
  if (typeof localStorage !== 'undefined') {
    persistUiBetaNoticeDismissed(localStorage);
  }
}
</script>

<style scoped>
.sd-beta-notice {
  margin: 0.75rem 0.75rem 0;
}

.sd-beta-notice__body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.sd-beta-notice__text {
  min-width: 0;
}

.sd-beta-notice__link {
  color: inherit;
  text-decoration: underline;
}

@media screen and (max-width: 767.9px) {
  .sd-beta-notice {
    margin: 0.5rem 0.5rem 0;
  }
}
</style>
