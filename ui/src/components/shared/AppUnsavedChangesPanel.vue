<script setup lang="ts">
import { activeUnsavedChangesSource, saveActiveUnsavedChanges } from '@/features/unsavedChanges';

async function handleSave() {
  await saveActiveUnsavedChanges();
}
</script>

<template>
  <transition name="unsaved-panel">
    <section v-if="activeUnsavedChangesSource" class="unsaved-panel">
      <div class="unsaved-panel-copy">
        <n-text class="unsaved-panel-title" tag="strong">
          {{ activeUnsavedChangesSource.label }} 有修改
        </n-text>
        <n-text depth="3" class="unsaved-panel-subtitle">
          不要忘记保存
        </n-text>
      </div>

      <n-button
        type="primary"
        class="unsaved-panel-action"
        :loading="activeUnsavedChangesSource.saving"
        :disabled="!activeUnsavedChangesSource.canSave"
        @click="handleSave"
      >
        <template #icon>
          <n-icon><i-carbon-save /></n-icon>
        </template>
        保存
      </n-button>
    </section>
  </transition>
</template>

<style scoped>
.unsaved-panel {
  display: flex;
  width: min(32rem, calc(100vw - 2rem));
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid rgba(248, 113, 113, 0.28);
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.9) 0%, rgba(255, 247, 237, 0.96) 100%);
  box-shadow:
    0 18px 40px rgba(15, 23, 42, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(14px);
  padding: 0.85rem 1rem;
  pointer-events: auto;
}

:global(.dark) .unsaved-panel {
  border-color: rgba(251, 191, 36, 0.28);
  background:
    linear-gradient(180deg, rgba(24, 33, 51, 0.92) 0%, rgba(30, 41, 59, 0.96) 100%);
  box-shadow:
    0 24px 52px rgba(2, 6, 23, 0.45),
    inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.unsaved-panel-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.15rem;
}

.unsaved-panel-title {
  color: var(--sd-text-primary);
  font-size: 0.96rem;
  line-height: 1.25;
}

.unsaved-panel-subtitle {
  font-size: 0.8rem;
}

.unsaved-panel-action {
  flex: 0 0 auto;
}

.unsaved-panel-enter-active,
.unsaved-panel-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.unsaved-panel-enter-from,
.unsaved-panel-leave-to {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

@media (max-width: 640px) {
  .unsaved-panel {
    width: min(100%, calc(100vw - 1rem));
    gap: 0.75rem;
    flex-wrap: wrap;
    align-items: stretch;
    border-radius: 14px;
    padding: 0.75rem 0.875rem;
  }

  .unsaved-panel-action {
    width: 100%;
  }
}
</style>
