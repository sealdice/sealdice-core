<template>
  <article
    :class="['sd-repeatable-item', { 'sd-repeatable-item--inline': inline }]"
    role="listitem"
  >
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
    inline?: boolean;
  }>(),
  {
    title: '',
    removable: true,
    removeLabel: '删除此项',
    showEnabled: false,
    enabled: true,
    enabledLabel: '启用此项',
    disabled: false,
    inline: false,
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

/* 单字段子项：标题、控件、删除操作收在一行，避免无意义的纵向留白。 */
.sd-repeatable-item--inline {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sd-space-md);
  padding-block: var(--sd-space-xs);
}

.sd-repeatable-item--inline .sd-repeatable-item__header {
  display: contents;
}

.sd-repeatable-item--inline .sd-repeatable-item__heading {
  grid-column: 1;
  grid-row: 1;
}

.sd-repeatable-item--inline .sd-repeatable-item__body {
  grid-column: 2;
  grid-row: 1;
}

.sd-repeatable-item--inline .sd-repeatable-item__actions {
  grid-column: 3;
  grid-row: 1;
  margin-left: 0;
}

@media (max-width: 640px) {
  .sd-repeatable-item__header {
    align-items: flex-start;
  }

  .sd-repeatable-item--inline {
    display: flex;
    align-items: stretch;
    flex-direction: column;
    gap: var(--sd-space-xs);
    padding-block: var(--sd-space-sm);
  }

  .sd-repeatable-item--inline .sd-repeatable-item__header {
    display: flex;
  }

  .sd-repeatable-item--inline .sd-repeatable-item__body {
    width: 100%;
  }
}
</style>
