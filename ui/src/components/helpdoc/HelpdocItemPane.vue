<template>
  <n-spin :show="loading">
    <main class="item-list-container">
      <header>
        <ProSearchForm
          :form="searchForm"
          :columns="searchColumns"
          size="small"
          label-width="72"
          label-placement="left"
          cols="1 s:2 l:4"
          :collapse-button-props="false"
        />
      </header>

      <n-data-table class="item-list" :columns="columns" :data="items" size="small" :bordered="false" remote :scroll-x="980" />

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
    </main>
  </n-spin>
</template>

<script setup lang="tsx">
import { computed, watch } from 'vue';
import { NFlex, NText, type DataTableColumns } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import type { HelpTextVo } from '@/api';
import {
  createDefaultHelpdocItemQuery,
  type HelpdocItemQueryModel,
} from '@/features/helpdoc/queries';
import {
  cloneSearchFormValues,
  overwriteSearchFormValues,
} from '@/features/searchForm/viewModel';

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
  { deep: true, immediate: true },
);
</script>

<style scoped>
.item-list-container {
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.item-list {
  width: 100%;
}

.item-list-pagination {
  margin-top: 10px;
}
</style>
