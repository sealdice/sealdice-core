<template>
  <div class="responsive-tabs">
    <ResponsiveDataView class="responsive-tabs__navigation" :compact-at="props.compactAt">
      <template #table>
        <n-tabs v-model:value="model" class="sd-scrollable-tabs" type="line">
          <n-tab
            v-for="option in props.options"
            :key="option.value"
            :name="option.value"
            :tab="option.label"
          />
        </n-tabs>
      </template>

      <template #compact>
        <n-select
          v-model:value="model"
          class="responsive-tabs__select"
          :options="props.options"
          aria-label="选择设置分类"
        />
      </template>
    </ResponsiveDataView>

    <section v-if="selectedOption" class="responsive-tabs__panel">
      <slot name="panel" :option="selectedOption" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
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

watch(
  () => props.options,
  options => {
    if (model.value && options.some(option => option.value === model.value)) return;
    model.value = options[0]?.value ?? '';
  }
);
</script>

<style scoped>
.responsive-tabs {
  display: grid;
  gap: var(--sd-page-section-gap);
  min-width: 0;
}

.responsive-tabs__navigation {
  container-type: inline-size;
}

.responsive-tabs__select {
  width: min(100%, 22rem);
}

.responsive-tabs__panel {
  min-width: 0;
}
</style>
