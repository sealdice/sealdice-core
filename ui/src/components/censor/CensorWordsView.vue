<template>
  <n-flex justify="space-between" class="censor-words-header">
    <h4>敏感词列表</h4>
    <n-flex align="center" class="censor-words-filter">
      <n-text v-if="filterCount > 0" type="info" class="text-xs">
        已过滤 {{ filterCount }} 条
      </n-text>
      <label class="censor-words-filter__field" for="censor-word-filter">
        <span>筛选敏感词</span>
        <n-input
          id="censor-word-filter"
          v-model:value="filter"
          size="small"
          placeholder="输入敏感词或匹配词"
          clearable
          aria-label="筛选敏感词"
        >
          <template #prefix>
            <n-icon><i-tabler-search /></n-icon>
          </template>
        </n-input>
      </label>
    </n-flex>
  </n-flex>

  <main class="mt-2 mb-8">
    <ResponsiveDataView :compact-at="520" aria-label="敏感词列表">
      <template #table>
        <n-data-table
          class="w-full"
          :columns="columns"
          :data="filteredWords"
          :scroll-x="480"
          virtual-scroll
        />
      </template>
      <template #compact>
        <ul class="censor-words-list">
          <li
            v-for="(word, index) in filteredWords"
            :key="`${word.main}-${index}`"
            class="censor-words-list__item"
          >
            <CensorSensitiveTag :level="word.level" />
            <div class="censor-words-list__tokens">
              <n-text
                v-for="related in word.related?.length ? word.related : [{ word: word.main }]"
                :key="related.word"
                class="censor-word-token"
              >
                {{ related.word }}
              </n-text>
            </div>
          </li>
        </ul>
      </template>
    </ResponsiveDataView>
  </main>
</template>

<script setup lang="tsx">
import { computed, ref } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import type { CensorWordItem } from '@/api';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';
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
          <n-flex size="small" wrap>
            {row.related.map(word => (
              <n-text key={word.word} class="censor-word-token">
                {word.word}
              </n-text>
            ))}
          </n-flex>
        );
      }
      return (
        <n-flex>
          <n-text class="censor-word-token">{row.main}</n-text>
        </n-flex>
      );
    },
  },
];
</script>

<style scoped>
:deep(.censor-word-token) {
  overflow-wrap: anywhere;
}

.censor-words-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.625rem;
  list-style: none;
}

.censor-words-list__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  border-bottom: 1px solid var(--sd-border-soft);
  gap: 0.75rem;
  padding: 0.75rem 0;
}

.censor-words-list__tokens {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 0.35rem 0.75rem;
}

.censor-words-filter__field {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

@media screen and (max-width: 639.9px) {
  .censor-words-header,
  .censor-words-filter {
    align-items: flex-start;
    flex-direction: column;
  }

  .censor-words-filter__field {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
