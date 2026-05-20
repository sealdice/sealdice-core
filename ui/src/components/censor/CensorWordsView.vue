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
    render: row => <CensorSensitiveTag level={row.level} />,
  },
  {
    key: 'related',
    title: '匹配词汇',
    render: row => {
      if (row.related?.length) {
        return (
          <n-flex size='small'>
            {row.related.map(word => (
              <n-text key={word.word}>{word.word}</n-text>
            ))}
          </n-flex>
        );
      }
      return (
        <n-flex>
          <n-text>{row.main}</n-text>
        </n-flex>
      );
    },
  },
];
</script>

<template>
  <n-flex justify="space-between">
    <h4>敏感词列表</h4>
    <n-flex align="center">
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
    <n-data-table class="w-full" :columns="columns" :data="filteredWords" virtual-scroll />
  </main>
</template>
