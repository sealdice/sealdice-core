<template>
  <div class="pending-action-row" :class="`pending-action-row--${action.kind}`">
    <n-icon class="pending-action-row__icon" :size="16" aria-hidden="true">
      <i-tabler-alert-triangle />
    </n-icon>

    <n-text class="pending-action-row__label">{{ statusText }}</n-text>

    <div class="pending-action-row__actions">
      <!-- 「放弃改动」使保存有对称的撤销出口，仅在来源提供 discard 时出现。 -->
      <n-button
        v-if="action.discard"
        secondary
        :size="size"
        :disabled="action.saving"
        @click="emit('discard')"
      >
        <template #icon>
          <n-icon><i-tabler-arrow-back-up /></n-icon>
        </template>
        放弃改动
      </n-button>

      <n-button
        type="primary"
        :size="size"
        :loading="action.saving"
        :disabled="!action.canSave || action.saving"
        @click="emit('run')"
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
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ActiveUnsavedChangesSource } from '@/features/unsavedChanges';

const props = withDefaults(
  defineProps<{
    action: ActiveUnsavedChangesSource;
    /** 是否显示来源名称。悬浮面板需要（用户已看不到标题），标题旁不需要。 */
    withLabel?: boolean;
    size?: 'small' | 'medium';
  }>(),
  {
    withLabel: false,
    size: 'medium',
  }
);

const emit = defineEmits<{ run: []; discard: [] }>();

const statusText = computed(() => {
  const state = props.action.kind === 'unsaved' ? '未保存' : '待重载';
  return props.withLabel ? `${props.action.label} ${state}` : state;
});
</script>

<style scoped>
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
