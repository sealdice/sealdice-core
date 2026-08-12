<template>
  <transition name="unsaved-panel">
    <section v-if="activeUnsavedChangesSource" class="unsaved-panel">
      <div class="unsaved-panel-copy">
        <n-text class="unsaved-panel-title" tag="strong">
          {{ activeUnsavedChangesSource.label }} 有修改
        </n-text>
        <n-text depth="3" class="unsaved-panel-subtitle"> 不要忘记保存 </n-text>
      </div>

      <n-button
        type="primary"
        class="unsaved-panel-action"
        :loading="activeUnsavedChangesSource.saving"
        :disabled="!activeUnsavedChangesSource.canSave"
        @click="handleSave"
      >
        <template #icon>
          <n-icon><i-tabler-device-floppy /></n-icon>
        </template>
        保存
      </n-button>
    </section>
  </transition>
</template>

<script setup lang="ts">
import { activeUnsavedChangesSource, saveActiveUnsavedChanges } from '@/features/unsavedChanges';

async function handleSave() {
  await saveActiveUnsavedChanges();
}
</script>

<style scoped>
.unsaved-panel {
  display: flex;
  width: min(32rem, calc(100vw - 2rem));
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--sd-warning-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  box-shadow: var(--sd-shadow-floating);
  padding: 0.85rem 1rem;
  pointer-events: auto;
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
    border-radius: var(--sd-radius-md);
    padding: 0.75rem 0.875rem;
  }

  .unsaved-panel-action {
    width: 100%;
  }
}
</style>
