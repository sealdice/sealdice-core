<template>
  <transition name="unsaved-panel">
    <section v-if="visible" class="unsaved-panel" role="status" aria-live="polite">
      <PendingActionRow
        v-for="action in actions"
        :key="action.scope"
        :action="action"
        with-label
        @run="handleRun(action.scope)"
      />
    </section>
  </transition>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import PendingActionRow from '@/components/shared/PendingActionRow.vue';
import {
  activePendingActions,
  isPendingActionAnchorVisible,
  runPendingAction,
} from '@/features/unsavedChanges';

const actions = computed(() => activePendingActions.value);

// 与 PageHeader 内的状态区互斥：锚点还在视口里时不出现，避免两处重复且相互遮挡。
const visible = computed(() => actions.value.length > 0 && !isPendingActionAnchorVisible.value);

async function handleRun(scope: string) {
  await runPendingAction(scope);
}
</script>

<style scoped>
.unsaved-panel {
  display: flex;
  /* 宽度贴合内容，避免出现远长于文案的空面板；上限防止长标签把面板拉满屏。 */
  width: fit-content;
  max-width: min(32rem, calc(100vw - 2rem));
  flex-direction: column;
  gap: var(--sd-space-sm);
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  box-shadow: var(--sd-shadow-floating);
  padding: 0.85rem 1rem;
  pointer-events: auto;
}

.unsaved-panel-enter-active,
.unsaved-panel-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.unsaved-panel-enter-from,
.unsaved-panel-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* 移动端沉底通栏：长表单里拇指可达，且不与顶部面包屑争夺空间。 */
@media (max-width: 767.9px) {
  .unsaved-panel {
    width: 100%;
    max-width: none;
    gap: var(--sd-space-xs);
    border: none;
    border-top: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-md) var(--sd-radius-md) 0 0;
    padding: var(--sd-space-sm) var(--sd-space-md);
    padding-bottom: calc(var(--sd-space-sm) + env(safe-area-inset-bottom, 0px));
  }

  .unsaved-panel-enter-from,
  .unsaved-panel-leave-to {
    transform: translateY(8px);
  }
}
</style>
