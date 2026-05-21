<script setup lang="tsx">
import { computed, ref } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import type { CensorWordItem } from '@/api';
import CensorSensitiveTag from './CensorSensitiveTag.vue';
import { filterCensorWords } from '@/features/censor/viewModel';

const props = defineProps<{
  words: CensorWordItem[];
}>();

const filter = ref('');
const filteredWords = computed(() => filterCensorWords(props.words, filter.value));
const filterCount = computed(() => props.words.length - filteredWords.value.length);

const columns: DataTableColumns<CensorWordItem> = [
  {
    key: 'level',
    title: '级别',
    minWidth: 110,
    render: row => <CensorSensitiveTag level={row.level} />,
  },
  {
    key: 'related',
    title: '匹配词汇',
    minWidth: 320,
    render: row => {
      if (row.related?.length) {
        return (
          <n-flex size='small' wrap>
            {row.related.map(word => (
              <n-text key={word.word} class='censor-word-token'>{word.word}</n-text>
            ))}
          </n-flex>
        );
      }
      return (
        <n-flex>
          <n-text class='censor-word-token'>{row.main}</n-text>
        </n-flex>
      );
    },
  },
];
</script>

<template>
  <n-flex justify="space-between" class="censor-words-header">
    <h4>敏感词列表</h4>
    <n-flex align="center" class="censor-words-filter">
      <n-text v-if="filterCount > 0" type="info" class="text-xs">
        已过滤 {{ filterCount }} 条
      </n-text>
      <span>
        <n-input v-model:value="filter" size="small" placeholder="" clearable>
          <template #prefix>
            <n-icon><i-carbon-search /></n-icon>
          </template>
        </n-input>
      </span>
    </n-flex>
  </n-flex>

  <main class="mt-2 mb-8">
    <n-data-table class="w-full" :columns="columns" :data="filteredWords" :scroll-x="480" virtual-scroll />
  </main>
</template>

<style scoped>
:deep(.censor-word-token) {
  overflow-wrap: anywhere;
}

@media screen and (max-width: 639.9px) {
  .censor-words-header,
  .censor-words-filter {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
