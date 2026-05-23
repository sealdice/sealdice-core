<script setup lang="ts">
import { computed } from 'vue';
import type { SelectOption } from 'naive-ui';
import type { ConfigItem } from '@/api';
import {
  normalizeTemplateValue,
} from '@/features/js/configModel';

const props = defineProps<{
  item: ConfigItem;
  pluginName: string;
  errorText?: string;
  checking?: boolean;
}>();

const emit = defineEmits<{
  change: [pluginName: string, key: string, value: unknown];
  reset: [pluginName: string, key: string];
  validate: [pluginName: string, key: string, value: string, type: string];
}>();

const type = computed(() => props.item.type ?? 'string');
const value = computed(() => props.item.value ?? props.item.defaultValue);
const isChanged = computed(() => JSON.stringify(value.value) !== JSON.stringify(props.item.defaultValue));
const optionItems = computed<SelectOption[]>(() => {
  if (!Array.isArray(props.item.option)) return [];
  return props.item.option.map(option => ({
    label: String(option),
    value: String(option),
  }));
});
const templateValues = computed(() => normalizeTemplateValue(value.value));

function updateValue(nextValue: unknown) {
  props.item.value = nextValue;
  emit('change', props.pluginName, props.item.key, nextValue);
}

function updateTaskValue(nextValue: string) {
  updateValue(nextValue);
  emit('validate', props.pluginName, props.item.key, nextValue, type.value);
}

function updateTemplateItem(index: number, nextValue: string) {
  const nextItems = [...templateValues.value];
  nextItems[index] = nextValue;
  updateValue(nextItems);
}

function addTemplateItem() {
  updateValue([...templateValues.value, '']);
}

function removeTemplateItem(index: number) {
  const nextItems = templateValues.value.filter((_, itemIndex) => itemIndex !== index);
  updateValue(nextItems.length ? nextItems : ['']);
}
</script>

<template>
  <section class="js-config-item">
    <header class="js-config-item__header">
      <div class="js-config-item__meta">
        <n-flex align="center" size="small" wrap>
          <n-text strong>{{ item.key }}</n-text>
          <n-tag v-if="item.deprecated" size="small" type="error" :bordered="false">
            废弃
          </n-tag>
          <n-tag v-if="type.startsWith('task:')" size="small" type="warning" :bordered="false">
            定时任务
          </n-tag>
        </n-flex>
        <n-text v-if="item.description" depth="3" class="js-config-item__description">
          {{ item.description }}
        </n-text>
      </div>
      <n-button
        v-if="isChanged"
        size="tiny"
        secondary
        @click="emit('reset', pluginName, item.key)"
      >
        重置
      </n-button>
    </header>

    <n-switch
      v-if="type === 'bool' || type === 'boolean'"
      size="small"
      :value="!!value"
      @update:value="updateValue"
    />

    <n-input-number
      v-else-if="type === 'int' || type === 'float' || type === 'number'"
      size="small"
      :precision="type === 'int' ? 0 : undefined"
      :value="Number(value) || 0"
      class="js-config-item__control js-config-item__control--short"
      @update:value="updateValue"
    />

    <n-select
      v-else-if="type === 'option'"
      size="small"
      :value="String(value ?? '')"
      :options="optionItems"
      class="js-config-item__control js-config-item__control--short"
      @update:value="updateValue"
    />

    <div v-else-if="type === 'template'" class="js-config-item__template">
      <div
        v-for="(templateValue, index) in templateValues"
        :key="`${item.key}-${index}`"
        class="js-config-item__template-row"
      >
        <n-input
          type="textarea"
          size="small"
          autosize
          :value="templateValue"
          @update:value="value => updateTemplateItem(index, value)"
        />
        <n-button
          v-if="index === 0"
          size="small"
          secondary
          circle
          @click="addTemplateItem"
        >
          <template #icon>
            <n-icon><i-carbon-add /></n-icon>
          </template>
        </n-button>
        <n-button
          v-else
          size="small"
          secondary
          circle
          type="error"
          @click="removeTemplateItem(index)"
        >
          <template #icon>
            <n-icon><i-carbon-close /></n-icon>
          </template>
        </n-button>
      </div>
    </div>

    <n-input
      v-else-if="type === 'task:cron' || type === 'task:daily'"
      type="textarea"
      size="small"
      autosize
      :status="errorText ? 'error' : undefined"
      :loading="checking"
      :value="String(value ?? '')"
      class="js-config-item__control"
      @update:value="updateTaskValue"
    />

    <n-input
      v-else
      size="small"
      :value="String(value ?? '')"
      class="js-config-item__control"
      @update:value="updateValue"
    />

    <n-text v-if="errorText" type="error" class="js-config-item__error">
      {{ errorText }}
    </n-text>
  </section>
</template>

<style scoped>
.js-config-item {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 12px 0;
}

.js-config-item + .js-config-item {
  border-top: 1px solid var(--sd-border-soft);
}

.js-config-item__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.js-config-item__meta {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.js-config-item__description {
  overflow-wrap: anywhere;
  font-size: 12px;
}

.js-config-item__control {
  max-width: 520px;
}

.js-config-item__control--short {
  width: min(100%, 240px);
}

.js-config-item__template {
  display: grid;
  max-width: 640px;
  gap: 8px;
}

.js-config-item__template-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: start;
}

.js-config-item__error {
  font-size: 12px;
}

@media (max-width: 640px) {
  .js-config-item__header {
    align-items: stretch;
    flex-direction: column;
  }

  .js-config-item__control {
    max-width: none;
  }
}
</style>
