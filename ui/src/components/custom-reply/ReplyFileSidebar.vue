<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import type { UploadCustomRequestOptions } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import type { FileInfo } from '@/api';
import { overwriteSearchFormValues } from '@/features/searchForm/viewModel';
import type { ReplyFileQuery } from '@/features/customReply/useCustomReplyEditor';

type ReplyFileSearchFormValues = Pick<ReplyFileQuery, 'keyword' | 'sortBy' | 'sortOrder'>;

const props = defineProps<{
  files: FileInfo[];
  total: number;
  selectedFilename: string;
  query: ReplyFileQuery;
  getFileEnableStatus: (filename: string, fallback: boolean) => boolean;
  formatUpdateTime: (ts: number) => string;
}>();

const emit = defineEmits<{
  select: [filename: string];
  create: [];
  openImport: [];
  delete: [];
  download: [];
  upload: [options: UploadCustomRequestOptions];
  updateQuery: [query: ReplyFileQuery];
}>();

const fileSortOptions = [
  { label: '按更新时间', value: 'updateTime' },
  { label: '按名称', value: 'name' },
];

const fileSortOrderOptions = [
  { label: '降序', value: 'desc' },
  { label: '升序', value: 'asc' },
];

const syncingFromProps = ref(false);
const searchForm = createProSearchForm<ReplyFileSearchFormValues>({
  initialValues: {
    keyword: '',
    sortBy: 'updateTime',
    sortOrder: 'desc',
  },
});

const searchColumns: ProSearchFormColumns<ReplyFileSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '按文件名搜索',
    },
  },
  {
    label: '排序字段',
    path: 'sortBy',
    field: 'select',
    fieldProps: {
      options: fileSortOptions,
    },
  },
  {
    label: '排序方向',
    path: 'sortOrder',
    field: 'select',
    fieldProps: {
      options: fileSortOrderOptions,
    },
  },
];

const page = computed({
  get: () => props.query.page,
  set: value => emit('updateQuery', { ...props.query, page: value }),
});

watch(
  () => [props.query.keyword, props.query.sortBy, props.query.sortOrder] as const,
  ([keyword, sortBy, sortOrder]) => {
    syncingFromProps.value = true;
    overwriteSearchFormValues(searchForm.values.value, { keyword, sortBy, sortOrder });
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
      ...props.query,
      keyword: values.keyword,
      sortBy: values.sortBy,
      sortOrder: values.sortOrder,
      page: 1,
    });
  },
  { deep: true },
);
</script>

<template>
  <aside class="reply-sidebar">
    <div class="panel-head">
      <div class="panel-title">
        <n-icon><i-carbon-folder /></n-icon>
        <span>文件管理</span>
      </div>
      <div class="panel-toolbar">
        <n-tooltip>
          <template #trigger>
            <n-button size="small" quaternary circle @click="emit('create')">
              <template #icon>
                <n-icon><i-carbon-document-add /></n-icon>
              </template>
            </n-button>
          </template>
          新建文件
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-upload
              action=""
              accept=".yaml"
              :custom-request="(options: UploadCustomRequestOptions) => emit('upload', options)"
              :show-file-list="false"
            >
              <n-button size="small" quaternary circle>
                <template #icon>
                  <n-icon><i-carbon-upload /></n-icon>
                </template>
              </n-button>
            </n-upload>
          </template>
          上传文件
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-button size="small" quaternary circle @click="emit('openImport')">
              <template #icon>
                <n-icon><i-carbon-document /></n-icon>
              </template>
            </n-button>
          </template>
          解析导入
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-button size="small" quaternary circle :disabled="!selectedFilename" @click="emit('download')">
              <template #icon>
                <n-icon><i-carbon-download /></n-icon>
              </template>
            </n-button>
          </template>
          下载文件
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-button size="small" quaternary circle type="error" :disabled="!selectedFilename" @click="emit('delete')">
              <template #icon>
                <n-icon><i-carbon-row-delete /></n-icon>
              </template>
            </n-button>
          </template>
          删除文件
        </n-tooltip>
      </div>
    </div>

    <div class="panel-controls">
      <ProSearchForm
        :form="searchForm"
        :columns="searchColumns"
        size="small"
        label-placement="left"
        label-width="72"
        cols="1"
        :show-suffix-grid-item="false"
        :collapse-button-props="false"
      />
    </div>

    <div class="panel-body">
      <n-empty v-if="!files.length" description="暂无文件" />
      <button
        v-for="item in files"
        :key="item.filename"
        type="button"
        class="file-item"
        :class="{ active: item.filename === selectedFilename }"
        @click="emit('select', item.filename)"
      >
        <div class="file-item-main">
          <div class="file-item-name-row">
            <n-icon class="file-item-icon"><i-carbon-document-blank /></n-icon>
            <span class="file-item-name">{{ item.filename }}</span>
          </div>
          <div class="file-item-meta">
            <n-tag size="tiny" :bordered="false" :type="getFileEnableStatus(item.filename, item.enable) ? 'success' : 'warning'">
              {{ getFileEnableStatus(item.filename, item.enable) ? '启用' : '停用' }}
            </n-tag>
            <span>{{ formatUpdateTime(item.updateTimestamp) }}</span>
            <span>{{ item.itemCount }} 条</span>
          </div>
        </div>
      </button>
    </div>

    <div class="panel-footer">
      <n-pagination
        v-model:page="page"
        :page-size="query.pageSize"
        :item-count="total"
        simple
      />
    </div>
  </aside>
</template>

<style scoped>
.reply-sidebar {
  display: flex;
  width: 280px;
  min-width: 240px;
  max-width: 320px;
  min-height: 0;
  flex-direction: column;
  border-right: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated-muted);
}

@supports (color: color-mix(in srgb, white, black)) {
  .reply-sidebar {
    background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 48%);
  }
}

.panel-head,
.panel-controls,
.panel-footer {
  border-bottom: 1px solid var(--sd-border-soft);
  padding: 0.75rem;
}

.panel-footer {
  border-top: 1px solid var(--sd-border-soft);
  border-bottom: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 700;
  line-height: 1;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-top: 0.5rem;
}

.panel-controls {
  min-width: 0;
}

.panel-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 0.5rem;
}

.file-item {
  display: block;
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  padding: 0.45rem 0.5rem;
  text-align: left;
}

.file-item:hover {
  background: var(--sd-bg-hover);
}

.file-item.active {
  background: var(--sd-bg-selected);
}

.file-item-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.35rem;
}

.file-item-name-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
}

.file-item-icon {
  flex: 0 0 auto;
}

.file-item-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
}

@media screen and (max-width: 1023.9px) {
  .reply-sidebar {
    width: 240px;
    min-width: 220px;
  }
}

@media screen and (max-width: 639.9px) {
  .reply-sidebar {
    width: 100%;
    max-width: none;
    border-right: 0;
    border-bottom: 1px solid var(--sd-border);
  }

  .panel-body {
    max-height: 220px;
  }
}
</style>
