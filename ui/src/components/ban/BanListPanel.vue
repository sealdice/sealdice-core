<template>
  <section class="ban-list-panel">
    <QueryToolbar :form="searchForm" :columns="searchColumns" :loading="loading" cols="1 s:2 l:3" />

    <ListActions>
      <n-button type="primary" secondary :loading="addPending" @click="emit('openAdd')">
        <template #icon>
          <n-icon><i-tabler-plus /></n-icon>
        </template>
        添加
      </n-button>
      <n-upload
        action=""
        accept=".json,application/json"
        :show-file-list="false"
        :custom-request="uploadBanFile"
      >
        <n-button type="primary" secondary :loading="importPending">
          <template #icon>
            <n-icon><i-tabler-upload /></n-icon>
          </template>
          导入
        </n-button>
      </n-upload>
      <n-button secondary @click="emit('export')">
        <template #icon>
          <n-icon><i-tabler-download /></n-icon>
        </template>
        导出
      </n-button>
    </ListActions>

    <n-spin :show="loading">
      <ListPanel>
        <n-list hoverable clickable class="ban-list-panel__list">
          <n-list-item v-for="item in items" :key="item.ID">
            <n-thing>
              <template #header>
                <n-flex size="small" align="center">
                  <n-tag :type="getBanRankMeta(item.rank).tagType" :bordered="false">
                    {{ getBanRankMeta(item.rank).label }}
                  </n-tag>
                  <n-text tag="strong">{{ item.ID }}</n-text>
                </n-flex>
              </template>
              <template #header-extra>
                <n-button type="error" size="small" secondary @click="emit('delete', item)">
                  <template #icon>
                    <n-icon><i-tabler-trash /></n-icon>
                  </template>
                  删除
                </n-button>
              </template>
              <template #description>
                <n-flex size="small" align="center" wrap>
                  <n-text>「{{ item.name || '未命名' }}」</n-text>
                  <n-text depth="3">怒气值：{{ item.score }}</n-text>
                </n-flex>
              </template>

              <n-flex vertical size="small" class="ban-list-panel__reasons">
                <div
                  v-for="(reason, index) in item.reasons ?? []"
                  :key="`${item.ID}-${index}`"
                  class="ban-list-panel__reason-item"
                >
                  <n-tooltip>
                    <template #trigger>
                      <n-tag size="small" :bordered="false">
                        {{ dayjs.unix(item.times?.[index] ?? item.banTime).fromNow() }}
                      </n-tag>
                    </template>
                    {{
                      dayjs.unix(item.times?.[index] ?? item.banTime).format('YYYY-MM-DD HH:mm:ss')
                    }}
                  </n-tooltip>
                  <n-text>
                    在 &lt;{{ item.places?.[index] || '未知地点' }}&gt;，原因：「{{ reason }}」
                  </n-text>
                </div>
              </n-flex>
            </n-thing>
          </n-list-item>
        </n-list>

        <n-empty
          v-if="!items.length"
          description="暂无黑白名单条目"
          class="ban-list-panel__empty"
        />
      </ListPanel>
    </n-spin>

    <footer class="ban-list-panel__footer">
      <n-pagination
        :page="query.page"
        :page-size="query.pageSize"
        :item-count="total"
        show-size-picker
        :page-sizes="[10, 20, 30, 50]"
        @update:page="updatePage"
        @update:page-size="updatePageSize"
      />
    </footer>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import dayjs from 'dayjs';
import type { UploadCustomRequestOptions } from 'naive-ui';
import { createProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import type { BanListInfoItem } from '@/api';
import { getBanRankMeta, type BanListQueryModel } from '@/features/ban/viewModel';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';
import QueryToolbar from '@/components/shared/QueryToolbar.vue';
import ListActions from '@/components/shared/ListActions.vue';
import ListPanel from '@/components/shared/ListPanel.vue';

const props = defineProps<{
  items: BanListInfoItem[];
  total: number;
  loading: boolean;
  query: BanListQueryModel;
  addPending: boolean;
  importPending: boolean;
}>();

const emit = defineEmits<{
  updateQuery: [patch: Partial<BanListQueryModel>];
  openAdd: [];
  delete: [item: BanListInfoItem];
  import: [file: File];
  export: [];
}>();

type BanSearchFormValues = Pick<BanListQueryModel, 'keyword' | 'ranks' | 'sortBy'>;

const defaultBanSearchFormValues = (): BanSearchFormValues => ({
  keyword: '',
  ranks: [-30, -10, 30, 0],
  sortBy: 'time',
});

const syncingFromProps = ref(false);

const searchForm = createProSearchForm<BanSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultBanSearchFormValues()),
});

const searchColumns: ProSearchFormColumns<BanSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '按 ID 或名字筛选',
    },
  },
  {
    label: '级别',
    path: 'ranks',
    field: 'select',
    fieldProps: {
      options: [
        { label: '拉黑', value: -30 },
        { label: '警告', value: -10 },
        { label: '信任', value: 30 },
        { label: '其它', value: 0 },
      ],
      multiple: true,
      flexProps: {
        wrap: true,
      },
    },
  },
  {
    label: '排序',
    path: 'sortBy',
    field: 'radio-group',
    fieldProps: {
      type: 'button',
      options: [
        { label: '按封禁时间', value: 'time' },
        { label: '按怒气值', value: 'score' },
      ],
      flexProps: {
        wrap: true,
      },
    },
  },
];

watch(
  () => [props.query.keyword, props.query.sortBy, props.query.ranks] as const,
  ([keyword, sortBy, ranks]) => {
    syncingFromProps.value = true;
    searchForm.values.value.keyword = keyword;
    searchForm.values.value.sortBy = sortBy;
    searchForm.values.value.ranks = ranks;
    void nextTick(() => {
      syncingFromProps.value = false;
    });
  },
  { deep: true, immediate: true }
);

watch(
  () => searchForm.values.value,
  values => {
    if (syncingFromProps.value) return;
    emit('updateQuery', {
      keyword: values.keyword,
      ranks: [...values.ranks],
      sortBy: values.sortBy,
      page: 1,
    });
  },
  { deep: true }
);

function updatePage(page: number) {
  emit('updateQuery', { page });
}

function updatePageSize(pageSize: number) {
  emit('updateQuery', { pageSize, page: 1 });
}

async function uploadBanFile(options: UploadCustomRequestOptions) {
  const file = options.file.file;
  if (!(file instanceof File)) {
    options.onError?.();
    return;
  }
  try {
    emit('import', file);
    options.onFinish?.();
  } catch {
    options.onError?.();
  }
}
</script>

<style scoped>
.ban-list-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.ban-list-panel__list {
  border-radius: 6px;
  background: var(--sd-bg-elevated);
}

.ban-list-panel__reasons {
  margin-top: 0.5rem;
}

.ban-list-panel__reason-item {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.ban-list-panel__empty {
  padding: 1.5rem 0;
}

.ban-list-panel__footer {
  display: flex;
  justify-content: center;
}
</style>
