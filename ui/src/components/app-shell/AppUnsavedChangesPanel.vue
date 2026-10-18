<template>
  <transition name="unsaved-panel">
    <section v-if="actions.length" class="unsaved-panel" role="status" aria-live="polite">
      <div v-for="action in actions" :key="action.scope" class="pending-action-row">
        <n-icon class="pending-action-row__icon" :size="16" aria-hidden="true">
          <i-tabler-alert-triangle />
        </n-icon>

        <n-text class="pending-action-row__label"
          >{{ action.label }} {{ action.kind === 'unsaved' ? '未保存' : '待重载' }}</n-text
        >

        <div class="pending-action-row__actions">
          <!-- 「放弃改动」使保存有对称的撤销出口，仅在来源提供 discard 时出现。 -->
          <n-button
            v-if="action.discard"
            secondary
            size="medium"
            :disabled="action.saving"
            @click="store.discardPendingAction(action.scope)"
          >
            <template #icon>
              <n-icon><i-tabler-arrow-back-up /></n-icon>
            </template>
            放弃改动
          </n-button>

          <n-button
            type="primary"
            size="medium"
            :loading="action.saving"
            :disabled="!action.canSave || action.saving"
            @click="store.runPendingAction(action.scope)"
          >
            <template #icon>
              <n-icon>
                <i-tabler-device-floppy v-if="action.kind === 'unsaved'" />
                <i-tabler-refresh v-else />
              </n-icon>
            </template>
            {{ action.actionText }}
          </n-button>
        </div>
      </div>
    </section>
  </transition>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useUnsavedChangesStore } from '@/features/unsavedChanges/store';

const store = useUnsavedChangesStore();
const { activePendingActions: actions } = storeToRefs(store);
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

.pending-action-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-xs);
}

/* 未保存与待重载同为 warning：即使保存成功，
   未重载时实际运行效果仍不符合用户预期。 */
.pending-action-row__icon {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  color: var(--sd-warning);
}

.pending-action-row__label {
  /* 文案占据剩余宽度，把操作按钮推到行尾，避免内容缩在悬浮面板左侧。 */
  flex: 1 1 auto;
  min-width: 0;
  color: var(--sd-text-secondary);
  font-size: 0.9rem;
  line-height: 1.3;
}

.pending-action-row__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: var(--sd-space-xs);
}

@media (max-width: 640px) {
  .pending-action-row {
    width: 100%;
    flex-wrap: wrap;
  }

  .pending-action-row__label {
    flex: 1 1 100%;
  }

  .pending-action-row__actions {
    width: 100%;
  }

  .pending-action-row__actions :deep(.n-button) {
    flex: 1 1 0;
    min-height: 44px;
  }
}
</style>
