<template>
  <ResponsiveDataView class="responsive-tabs" :compact-at="props.compactAt">
    <template #table>
      <n-tabs v-model:value="model" type="line" animated>
        <n-tab-pane
          v-for="option in props.options"
          :key="option.value"
          :name="option.value"
          :tab="option.label"
        >
          <slot name="panel" :option="option" />
        </n-tab-pane>
      </n-tabs>
    </template>

    <template #compact>
      <div class="responsive-tabs__compact">
        <n-select
          v-model:value="model"
          class="responsive-tabs__select"
          :options="props.options"
          aria-label="选择设置分类"
        />
        <section v-if="selectedOption" class="responsive-tabs__panel">
          <slot name="panel" :option="selectedOption" />
        </section>
      </div>
    </template>
  </ResponsiveDataView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { SelectOption } from 'naive-ui';
import ResponsiveDataView from './ResponsiveDataView.vue';

export type ResponsiveTabOption = SelectOption & { label: string; value: string };

const props = withDefaults(
  defineProps<{
    options: ResponsiveTabOption[];
    compactAt?: number;
  }>(),
  {
    compactAt: 760,
  }
);

const model = defineModel<string>('value', { required: true });

defineSlots<{
  panel(props: { option: ResponsiveTabOption }): unknown;
}>();

const selectedOption = computed(
  () => props.options.find(option => option.value === model.value) ?? props.options[0]
);
</script>

<style scoped>
.responsive-tabs {
  container-type: inline-size;
}

.responsive-tabs__compact {
  display: grid;
  gap: var(--sd-page-section-gap);
}

.responsive-tabs :deep(.n-tab-pane) {
  padding-top: var(--sd-page-section-gap);
}

.responsive-tabs__select {
  width: min(100%, 22rem);
}

.responsive-tabs__panel {
  min-width: 0;
}
</style>
