<template>
  <div class="pending-action-row" :class="`pending-action-row--${action.kind}`">
    <n-icon class="pending-action-row__icon" :size="16" aria-hidden="true">
      <i-tabler-alert-triangle v-if="action.kind === 'unsaved'" />
      <i-tabler-refresh-alert v-else />
    </n-icon>

    <n-text class="pending-action-row__label">{{ statusText }}</n-text>

    <n-button
      type="primary"
      class="pending-action-row__button"
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

const emit = defineEmits<{ run: [] }>();

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

.pending-action-row__icon {
  flex: 0 0 auto;
}

.pending-action-row--unsaved .pending-action-row__icon {
  color: var(--sd-warning);
}

.pending-action-row--reload .pending-action-row__icon {
  color: var(--sd-info);
}

.pending-action-row__label {
  min-width: 0;
  margin-right: var(--sd-space-2xs);
  color: var(--sd-text-secondary);
  font-size: 0.9rem;
  line-height: 1.3;
}

.pending-action-row__button {
  flex: 0 0 auto;
}

@media (max-width: 640px) {
  .pending-action-row {
    width: 100%;
    flex-wrap: wrap;
  }

  .pending-action-row__label {
    flex: 1 1 auto;
  }

  .pending-action-row__button {
    width: 100%;
    min-height: 44px;
  }
}
</style>
