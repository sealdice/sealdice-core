<script setup lang="ts">
import type { BaseSettingSearchEntry } from '@/features/baseSetting/viewModel';

const keyword = defineModel<string>('keyword', { required: true });

defineProps<{
  results: BaseSettingSearchEntry[];
}>();

const emit = defineEmits<{
  select: [entry: BaseSettingSearchEntry];
}>();
</script>

<template>
  <section class="base-setting-search">
    <n-input v-model:value="keyword" clearable placeholder="搜索配置项、功能名、提示词">
      <template #prefix>
        <n-icon><i-carbon-search /></n-icon>
      </template>
    </n-input>

    <n-card v-if="keyword.trim() && results.length" size="small" class="search-result-card">
      <n-space vertical size="small">
        <button
          v-for="entry in results"
          :key="entry.fieldId"
          type="button"
          class="search-result-item"
          @click="emit('select', entry)"
        >
          <strong>{{ entry.label }}</strong>
          <span>{{ entry.tabTitle }} / {{ entry.groupTitle }}</span>
        </button>
      </n-space>
    </n-card>
    <n-text v-else-if="keyword.trim()" depth="3" class="search-empty">
      没有找到匹配的配置项
    </n-text>
  </section>
</template>

<style scoped>
.base-setting-search {
  position: relative;
  margin-bottom: 1rem;
}

.search-result-card {
  margin-top: 0.5rem;
}

.search-result-item {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 0;
  background: transparent;
  padding: 0.5rem 0.25rem;
  text-align: left;
  cursor: pointer;
}

.search-result-item:hover {
  background: var(--sd-bg-hover);
}

.search-result-item span,
.search-empty {
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}
</style>
