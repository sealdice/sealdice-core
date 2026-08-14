<template>
  <n-spin :show="loading">
    <ListWorkspace class="item-list-container">
      <QueryToolbar :form="searchForm" :columns="searchColumns" cols="1 s:2 l:4" />

      <ListPanel>
        <ResponsiveDataView :compact-at="900" aria-label="帮助文档词条">
          <template #table>
            <n-data-table
              class="item-list"
              :columns="columns"
              :data="items"
              size="small"
              :bordered="false"
              remote
              :scroll-x="1210"
            />
          </template>
          <template #compact>
            <ul class="item-compact-list">
              <li v-for="item in items" :key="item.id" class="item-compact-list__item">
                <div class="item-compact-list__heading">
                  <strong>{{ item.title }}</strong>
                  <span>#{{ item.id }}</span>
                </div>
                <n-flex size="small" wrap>
                  <n-tag size="small" :bordered="false">
                    {{ item.group || '未分组' }}
                  </n-tag>
                  <n-text depth="3">{{ item.packageName || item.from || '未知来源' }}</n-text>
                </n-flex>
                <p>{{ item.content || '暂无内容' }}</p>
              </li>
            </ul>
          </template>
        </ResponsiveDataView>
      </ListPanel>

      <footer>
        <n-flex class="item-list-pagination" align="center" justify="end" wrap>
          <n-text depth="3">共 {{ total }} 条</n-text>
          <n-pagination
            v-model:page="query.pageNum"
            v-model:page-size="query.pageSize"
            show-size-picker
            show-quick-jumper
            :page-sizes="[10, 20, 30, 50]"
            :page-slot="5"
            :item-count="total"
          />
        </n-flex>
      </footer>
    </ListWorkspace>
  </n-spin>
</template>

<script setup lang="tsx">
import { computed, watch } from 'vue';
import { NFlex, NText, type DataTableColumns } from 'naive-ui';
import { createProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import QueryToolbar from '@/components/shared/QueryToolbar.vue';
import ListPanel from '@/components/shared/ListPanel.vue';
import ListWorkspace from '@/components/shared/ListWorkspace.vue';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';
import type { HelpTextVo } from '@/api';
import {
  createDefaultHelpdocItemQuery,
  type HelpdocItemQueryModel,
} from '@/features/helpdoc/queries';
import { cloneSearchFormValues, overwriteSearchFormValues } from '@/features/searchForm/viewModel';

const query = defineModel<HelpdocItemQueryModel>('query', { required: true });

const props = defineProps<{
  loading: boolean;
  items: HelpTextVo[];
  total: number;
  groupOptions: { label: string; value: string }[];
  columns: DataTableColumns<HelpTextVo>;
}>();

const emit = defineEmits<{
  search: [];
  reset: [];
}>();

type HelpdocSearchFormValues = Pick<HelpdocItemQueryModel, 'id' | 'group' | 'from' | 'title'>;

const defaultHelpdocSearchFormValues = (): HelpdocSearchFormValues => {
  const defaults = createDefaultHelpdocItemQuery();
  return {
    id: defaults.id,
    group: defaults.group,
    from: defaults.from,
    title: defaults.title,
  };
};

const searchForm = createProSearchForm<HelpdocSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultHelpdocSearchFormValues()),
  onSubmit: values => {
    Object.assign(query.value, values, { pageNum: 1 });
    emit('search');
  },
  onReset: () => {
    emit('reset');
  },
});

const searchColumns = computed<ProSearchFormColumns<HelpdocSearchFormValues>>(() => [
  {
    label: '序号',
    path: 'id',
    field: 'digit',
    fieldProps: {
      clearable: true,
    },
  },
  {
    label: '分组',
    path: 'group',
    field: 'select',
    fieldProps: {
      options: props.groupOptions,
      placeholder: '选择分组',
      filterable: true,
      clearable: true,
    },
  },
  {
    label: '来源文件',
    path: 'from',
    field: 'input',
    fieldProps: {
      clearable: true,
    },
  },
  {
    label: '词条名',
    path: 'title',
    field: 'input',
    fieldProps: {
      clearable: true,
    },
  },
]);

watch(
  query,
  next => {
    overwriteSearchFormValues(searchForm.values.value, {
      id: next.id,
      group: next.group,
      from: next.from,
      title: next.title,
    });
  },
  { deep: true, immediate: true }
);
</script>

<style scoped>
.item-list {
  width: 100%;
}

.item-compact-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.625rem;
  list-style: none;
}

.item-compact-list__item {
  display: grid;
  min-width: 0;
  border-bottom: 1px solid var(--sd-border-soft);
  gap: 0.55rem;
  padding: 0.75rem 0;
}

.item-compact-list__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.item-compact-list__heading strong,
.item-compact-list__item p {
  overflow-wrap: anywhere;
}

.item-compact-list__heading span {
  color: var(--sd-text-muted);
  flex: 0 0 auto;
  font-size: 0.78rem;
}

.item-compact-list__item p {
  display: -webkit-box;
  margin: 0;
  color: var(--sd-text-secondary);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
  white-space: pre-wrap;
}
</style>
