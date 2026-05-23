<script setup lang="tsx">
import { computed, nextTick, ref, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { filesize } from 'filesize';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import { NButton, NSpace, NTag, type DataTableColumns, type UploadCustomRequestOptions, useMessage } from 'naive-ui';
import type { ResourceItem } from '@/api';
import ResourcePreview from '@/components/resource/ResourcePreview.vue';
import {
  formatResourcePageSummary,
  formatResourceTypeLabel,
  getResourceKey,
  getResourceTypeTagType,
  isResourceUploadFileAccepted,
  RESOURCE_PAGE_SIZE_OPTIONS,
  type ResourceListQueryModel,
} from '@/features/resource/viewModel';
import {
  cloneSearchFormValues,
  overwriteSearchFormValues,
} from '@/features/searchForm/viewModel';

const props = defineProps<{
  items: ResourceItem[];
  total: number;
  loading: boolean;
  query: ResourceListQueryModel;
  uploadPending: boolean;
  deletingPath: string;
  downloadingPath: string;
}>();

const emit = defineEmits<{
  updateQuery: [patch: Partial<ResourceListQueryModel>];
  upload: [file: File];
  copy: [item: ResourceItem];
  download: [item: ResourceItem];
  delete: [item: ResourceItem];
  detail: [item: ResourceItem];
  refresh: [];
}>();

type ResourceSearchFormValues = Pick<ResourceListQueryModel, 'keyword' | 'sortBy' | 'sortOrder'>;

const message = useMessage();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');
const pageSizeOptions = [...RESOURCE_PAGE_SIZE_OPTIONS];
const syncingFromProps = ref(false);

const defaultResourceSearchFormValues = (): ResourceSearchFormValues => ({
  keyword: '',
  sortBy: 'name',
  sortOrder: 'asc',
});

const searchForm = createProSearchForm<ResourceSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultResourceSearchFormValues()),
});

const searchColumns: ProSearchFormColumns<ResourceSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索文件名 / 路径 / 扩展名',
    },
  },
  {
    label: '排序字段',
    path: 'sortBy',
    field: 'select',
    fieldProps: {
      options: [
        { label: '名称', value: 'name' },
        { label: '大小', value: 'size' },
        { label: '扩展名', value: 'ext' },
        { label: '路径', value: 'path' },
      ],
    },
  },
  {
    label: '排序方向',
    path: 'sortOrder',
    field: 'radio-group',
    fieldProps: {
      type: 'button',
      options: [
        { label: '升序', value: 'asc' },
        { label: '降序', value: 'desc' },
      ],
      flexProps: {
        wrap: true,
      },
    },
  },
];

const summary = computed(() => formatResourcePageSummary({
  total: props.total,
  page: props.query.page,
  pageSize: props.query.pageSize,
}));

const columns = computed<DataTableColumns<ResourceItem>>(() => [
  {
    title: '预览',
    key: 'preview',
    width: 96,
    render: row => <ResourcePreview item={row} thumbnail />,
  },
  {
    title: '文件',
    key: 'name',
    minWidth: 280,
    render: row => (
      <div class='resource-list-panel__file-cell'>
        <strong>{row.name}</strong>
        <span>{row.path}</span>
      </div>
    ),
  },
  {
    title: '类型',
    key: 'type',
    width: 96,
    render: row => (
      <NTag size='small' bordered={false} type={getResourceTypeTagType(row.type)}>
        {formatResourceTypeLabel(row.type)}
      </NTag>
    ),
  },
  {
    title: '大小',
    key: 'size',
    width: 112,
    render: row => <NTag size='small' bordered={false}>{filesize(row.size)}</NTag>,
  },
  {
    title: '操作',
    key: 'actions',
    width: 300,
    render: row => (
      <NSpace justify='end' size='small'>
        <NButton size='small' secondary type='info' onClick={() => emit('copy', row)}>
          复制海豹码
        </NButton>
        <NButton size='small' secondary onClick={() => emit('detail', row)}>
          详情
        </NButton>
        <NButton
          size='small'
          secondary
          type='success'
          loading={props.downloadingPath === row.path}
          onClick={() => emit('download', row)}
        >
          下载
        </NButton>
        <NButton
          size='small'
          secondary
          type='error'
          loading={props.deletingPath === row.path}
          onClick={() => emit('delete', row)}
        >
          删除
        </NButton>
      </NSpace>
    ),
  },
]);

watch(
  () => [props.query.keyword, props.query.sortBy, props.query.sortOrder] as const,
  ([keyword, sortBy, sortOrder]) => {
    syncingFromProps.value = true;
    overwriteSearchFormValues(searchForm.values.value, {
      keyword,
      sortBy,
      sortOrder,
    });
    void nextTick(() => {
      syncingFromProps.value = false;
    });
  },
  { immediate: true },
);

watch(
  () => searchForm.values.value,
  values => {
    if (syncingFromProps.value) return;
    emit('updateQuery', {
      keyword: values.keyword,
      sortBy: values.sortBy,
      sortOrder: values.sortOrder,
      page: 1,
    });
  },
  { deep: true },
);

function updatePage(page: number) {
  emit('updateQuery', { page });
}

function updatePageSize(pageSize: number) {
  emit('updateQuery', { pageSize, page: 1 });
}

async function uploadResourceFile(options: UploadCustomRequestOptions) {
  const file = options.file.file;
  if (!(file instanceof File)) {
    options.onError?.();
    return;
  }
  if (!isResourceUploadFileAccepted(file)) {
    message.error('仅支持 PNG、JPG/JPEG、GIF 图片');
    options.onError?.();
    return;
  }

  emit('upload', file);
  options.onFinish?.();
}
</script>

<template>
  <section class="resource-list-panel">
    <header class="resource-list-panel__toolbar">
      <ProSearchForm
        :form="searchForm"
        :columns="searchColumns"
        size="small"
        label-placement="left"
        label-width="78"
        cols="1 m:2 xl:3"
        :show-suffix-grid-item="false"
        :collapse-button-props="false"
      />

      <n-flex align="center" justify="end" wrap>
        <n-button secondary :loading="loading" @click="emit('refresh')">
          <template #icon>
            <n-icon><i-carbon-renew /></n-icon>
          </template>
          刷新
        </n-button>
        <n-upload
          action=""
          multiple
          accept=".png,.jpg,.jpeg,.gif,image/png,image/jpeg,image/gif"
          :show-file-list="false"
          :custom-request="uploadResourceFile"
        >
          <n-button type="primary" :loading="uploadPending">
            <template #icon>
              <n-icon><i-carbon-upload /></n-icon>
            </template>
            上传图片
          </n-button>
        </n-upload>
      </n-flex>
    </header>

    <n-spin :show="loading && isMobile">
      <div v-if="isMobile" class="resource-list-panel__cards">
        <article v-for="item in items" :key="getResourceKey(item)" class="resource-list-panel__card">
          <button class="resource-list-panel__preview-button" type="button" @click="emit('detail', item)">
            <ResourcePreview :item="item" thumbnail />
          </button>
          <div class="resource-list-panel__card-main">
            <div class="resource-list-panel__card-title">
              <strong>{{ item.name }}</strong>
              <n-tag size="small" :bordered="false" :type="getResourceTypeTagType(item.type)">
                {{ formatResourceTypeLabel(item.type) }}
              </n-tag>
            </div>
            <n-text code class="resource-list-panel__path">{{ item.path }}</n-text>
            <n-flex align="center" justify="space-between" wrap>
              <n-tag size="small" :bordered="false">{{ filesize(item.size) }}</n-tag>
              <n-flex size="small" justify="end">
                <n-button size="tiny" secondary type="info" @click="emit('copy', item)">
                  复制码
                </n-button>
                <n-button size="tiny" secondary @click="emit('detail', item)">
                  详情
                </n-button>
                <n-button
                  size="tiny"
                  secondary
                  type="success"
                  :loading="downloadingPath === item.path"
                  @click="emit('download', item)"
                >
                  下载
                </n-button>
                <n-button
                  size="tiny"
                  secondary
                  type="error"
                  :loading="deletingPath === item.path"
                  @click="emit('delete', item)"
                >
                  删除
                </n-button>
              </n-flex>
            </n-flex>
          </div>
        </article>
      </div>

      <n-data-table
        v-else
        :columns="columns"
        :data="items"
        :loading="loading"
        :bordered="false"
        :row-key="getResourceKey"
        :scroll-x="900"
        size="small"
      />

      <n-empty v-if="!loading && items.length === 0" description="暂无图片资源" class="resource-list-panel__empty">
        <template #extra>
          <n-text depth="3">上传图片后可在骰子消息中使用 [图:路径] 引用。</n-text>
        </template>
      </n-empty>
    </n-spin>

    <footer class="resource-list-panel__footer">
      <n-text depth="3">{{ summary }}</n-text>
      <n-pagination
        :page="query.page"
        :page-size="query.pageSize"
        :item-count="total"
        show-size-picker
        :page-sizes="pageSizeOptions"
        @update:page="updatePage"
        @update:page-size="updatePageSize"
      />
    </footer>
  </section>
</template>

<style scoped>
.resource-list-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.resource-list-panel__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  align-items: start;
}

:deep(.resource-list-panel__file-cell) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

:deep(.resource-list-panel__file-cell strong) {
  overflow-wrap: anywhere;
  font-weight: 650;
}

:deep(.resource-list-panel__file-cell span) {
  overflow-wrap: anywhere;
  color: var(--sd-text-muted);
  font-size: 12px;
}

.resource-list-panel__cards {
  display: grid;
  gap: 12px;
}

.resource-list-panel__card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-elevated);
}

.resource-list-panel__preview-button {
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.resource-list-panel__card-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
}

.resource-list-panel__card-title {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.resource-list-panel__card-title strong {
  overflow-wrap: anywhere;
  font-weight: 650;
}

.resource-list-panel__path {
  overflow-wrap: anywhere;
}

.resource-list-panel__empty {
  padding: 40px 0;
}

.resource-list-panel__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

@media (max-width: 1080px) {
  .resource-list-panel__toolbar {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .resource-list-panel__footer {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
