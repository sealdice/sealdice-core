<template>
  <section class="sd-repeatable-list">
    <div v-if="$slots.default" class="sd-repeatable-list__items" role="list">
      <slot />
    </div>
    <n-empty v-else-if="emptyText" :description="emptyText" size="small" />

    <div class="sd-repeatable-list__footer">
      <slot name="footer">
        <n-button
          type="primary"
          secondary
          size="small"
          :disabled="addDisabled"
          @click="emit('add')"
        >
          <template #icon>
            <n-icon><i-tabler-plus /></n-icon>
          </template>
          {{ addLabel }}
        </n-button>
      </slot>
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    addLabel?: string;
    addDisabled?: boolean;
    emptyText?: string;
  }>(),
  {
    addLabel: '添加一项',
    addDisabled: false,
    emptyText: '',
  }
);

const emit = defineEmits<{
  add: [];
}>();
</script>

<style scoped>
.sd-repeatable-list {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-space-sm);
}

.sd-repeatable-list__items {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-space-sm);
}

.sd-repeatable-list__footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sd-space-xs);
}
</style>
