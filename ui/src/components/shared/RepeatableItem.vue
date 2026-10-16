<template>
  <article class="sd-repeatable-item" role="listitem">
    <header
      v-if="title || showEnabled || removable || $slots.leading || $slots.actions"
      class="sd-repeatable-item__header"
    >
      <div class="sd-repeatable-item__heading">
        <slot name="leading" />
        <n-text v-if="title" strong>{{ title }}</n-text>
      </div>

      <div class="sd-repeatable-item__actions">
        <n-switch
          v-if="showEnabled"
          :value="enabled"
          size="small"
          :disabled="disabled"
          :aria-label="enabledLabel"
          @update:value="emit('update:enabled', Boolean($event))"
        />
        <slot name="actions" />
        <n-tooltip v-if="removable">
          <template #trigger>
            <n-button
              quaternary
              circle
              size="small"
              type="error"
              :disabled="disabled"
              :aria-label="removeLabel"
              @click="emit('remove')"
            >
              <template #icon>
                <n-icon><i-tabler-trash /></n-icon>
              </template>
            </n-button>
          </template>
          {{ removeLabel }}
        </n-tooltip>
      </div>
    </header>

    <div class="sd-repeatable-item__body">
      <slot />
    </div>
  </article>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string;
    removable?: boolean;
    removeLabel?: string;
    showEnabled?: boolean;
    enabled?: boolean;
    enabledLabel?: string;
    disabled?: boolean;
  }>(),
  {
    title: '',
    removable: true,
    removeLabel: '删除此项',
    showEnabled: false,
    enabled: true,
    enabledLabel: '启用此项',
    disabled: false,
  }
);

const emit = defineEmits<{
  remove: [];
  'update:enabled': [value: boolean];
}>();
</script>

<style scoped>
.sd-repeatable-item {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-space-sm);
  padding: var(--sd-space-sm);
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-elevated-soft);
}

.sd-repeatable-item__header {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: var(--sd-space-sm);
}

.sd-repeatable-item__heading,
.sd-repeatable-item__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-xs);
}

.sd-repeatable-item__actions {
  margin-left: auto;
}

.sd-repeatable-item__body {
  min-width: 0;
}

@media (max-width: 640px) {
  .sd-repeatable-item__header {
    align-items: flex-start;
  }
}
</style>
