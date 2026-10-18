<template>
  <aside class="reply-sidebar" :class="{ 'is-mobile-expanded': mobileExpanded }">
    <div class="panel-head">
      <div class="panel-title">
        <n-icon><i-tabler-folder /></n-icon>
        <span>文件管理</span>
      </div>
      <div class="panel-toolbar">
        <n-tooltip>
          <template #trigger>
            <n-button size="small" quaternary circle @click="emit('create')">
              <template #icon>
                <n-icon><i-tabler-file-plus /></n-icon>
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
                  <n-icon><i-tabler-upload /></n-icon>
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
                <n-icon><i-tabler-file-text /></n-icon>
              </template>
            </n-button>
          </template>
          解析导入
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-button
              size="small"
              quaternary
              circle
              :disabled="!selectedFilename"
              @click="emit('download')"
            >
              <template #icon>
                <n-icon><i-tabler-download /></n-icon>
              </template>
            </n-button>
          </template>
          下载文件
        </n-tooltip>

        <n-tooltip>
          <template #trigger>
            <n-button
              size="small"
              quaternary
              circle
              type="error"
              :disabled="!selectedFilename"
              @click="emit('delete')"
            >
              <template #icon>
                <n-icon><i-tabler-trash /></n-icon>
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

    <div class="file-selection">
      <button
        type="button"
        class="file-selection-trigger"
        :aria-expanded="mobileExpanded"
        aria-controls="reply-sidebar-file-list"
        @click="toggleMobileExpanded"
      >
        <n-icon class="file-selection-icon"><i-tabler-file-text /></n-icon>
        <span class="file-selection-current">
          <template v-if="selectedFilename">{{ selectedFilename }}</template>
          <template v-else>选择回复文件</template>
        </span>
        <span class="file-selection-count">共 {{ total }} 个</span>
        <span class="panel-toggle-text">{{ mobileExpanded ? '收起' : '展开' }}</span>
        <n-icon class="panel-chevron"><i-tabler-chevron-down /></n-icon>
      </button>

      <div id="reply-sidebar-file-list" class="file-selection-list">
        <div class="panel-body">
          <ListEmptyState v-if="!files.length" description="暂无回复文件" />
          <button
            v-for="item in files"
            :key="item.filename"
            type="button"
            class="file-item"
            :class="{ active: item.filename === selectedFilename }"
            @click="selectFile(item.filename)"
          >
            <div class="file-item-main">
              <div class="file-item-name-row">
                <n-icon class="file-item-icon"><i-tabler-file-text /></n-icon>
                <span class="file-item-name">{{ item.filename }}</span>
              </div>
              <div class="file-item-meta">
                <n-tag
                  size="tiny"
                  :bordered="false"
                  :type="getFileEnableStatus(item.filename, item.enable) ? 'success' : 'warning'"
                >
                  {{ getFileEnableStatus(item.filename, item.enable) ? '启用' : '停用' }}
                </n-tag>
                <n-tag v-if="item.packageId" size="tiny" :bordered="false">
                  来源包 {{ item.packageId }}
                </n-tag>
                <span>{{ formatUpdateTime(item.updateTimestamp) }}</span>
                <span>{{ item.itemCount }} 条</span>
              </div>
            </div>
          </button>
        </div>

        <div
          v-if="shouldShowListPagination({ total, page: query.page, pageSize: query.pageSize })"
          class="panel-footer"
        >
          <n-pagination
            v-model:page="page"
            :page-size="query.pageSize"
            :item-count="total"
            simple
          />
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import type { UploadCustomRequestOptions } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import type { FileInfo } from '@/api';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import { shouldShowListPagination } from '@/components/shared/listPagination';
import { overwriteSearchFormValues } from '@/features/searchForm/viewModel';
import type { ReplyFileQuery } from '@/features/customReply/useCustomReplyEditor';

type ReplyFileSearchFormValues = Pick<ReplyFileQuery, 'keyword' | 'sortBy' | 'sortOrder'>;
type ReplySidebarFile = FileInfo & { packageId?: string };

const props = defineProps<{
  files: ReplySidebarFile[];
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
const mobileExpanded = ref(false);

function toggleMobileExpanded() {
  mobileExpanded.value = !mobileExpanded.value;
}

function selectFile(filename: string) {
  emit('select', filename);
}

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
  () => props.selectedFilename,
  () => {
    mobileExpanded.value = false;
  }
);

watch(
  () => [props.query.keyword, props.query.sortBy, props.query.sortOrder] as const,
  ([keyword, sortBy, sortOrder]) => {
    syncingFromProps.value = true;
    overwriteSearchFormValues(searchForm.values.value, { keyword, sortBy, sortOrder });
    void nextTick(() => {
      syncingFromProps.value = false;
    });
  },
  { immediate: true }
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
  { deep: true }
);
</script>

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

.panel-chevron,
.panel-toggle-text {
  display: none;
}

.file-selection {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
}

.file-selection-trigger {
  display: none;
}

.file-selection-list {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-top: 0.5rem;
}

.panel-toolbar :deep(.n-upload) {
  width: auto;
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

@container reply-editor (max-width: 720px) {
  .reply-sidebar {
    width: 100%;
    min-width: 0;
    max-width: none;
    border-right: 0;
    border-bottom: 1px solid var(--sd-border);
  }

  .panel-head {
    padding: 0.625rem 0.75rem;
  }

  .panel-toolbar {
    flex-wrap: wrap;
    margin-top: 0.4rem;
  }

  .panel-controls {
    padding: 0.625rem 0.75rem;
  }

  .file-selection {
    gap: 0.5rem;
    padding: 0.5rem;
  }

  .file-selection-trigger {
    display: flex;
    min-height: 44px;
    align-items: center;
    gap: 0.5rem;
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-md);
    background: var(--sd-bg-elevated);
    color: inherit;
    font: inherit;
    line-height: 1;
    padding: 0 0.75rem;
    text-align: left;
    cursor: pointer;
  }

  .file-selection-icon,
  .file-selection-count,
  .panel-toggle-text,
  .panel-chevron {
    flex: 0 0 auto;
  }

  .file-selection-current {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-selection-count {
    color: var(--sd-text-muted);
    font-size: 0.78rem;
    font-weight: 400;
  }

  .panel-chevron,
  .panel-toggle-text {
    display: inline-flex;
    align-items: center;
  }

  .panel-toggle-text {
    color: var(--sd-text-secondary);
    font-size: 0.8rem;
    font-weight: 400;
  }

  .panel-chevron {
    transition: transform var(--sd-transition-fast);
  }

  .reply-sidebar.is-mobile-expanded .panel-chevron {
    transform: rotate(180deg);
  }

  .file-selection-list {
    display: none;
  }

  .reply-sidebar.is-mobile-expanded .file-selection-list {
    display: flex;
    min-height: 0;
    flex-direction: column;
  }

  .panel-body {
    max-height: 220px;
  }
}
</style>
