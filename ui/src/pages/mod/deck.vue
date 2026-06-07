<template>
  <main class="deck-page">
    <header class="page-header">
      <n-button type="primary" :loading="reloadMutation.isPending.value" @click="doReload">
        <template #icon>
          <n-icon><i-carbon-renew /></n-icon>
        </template>
        重载牌堆
      </n-button>
    </header>

    <n-spin :show="pageBusy">
      <section class="deck-search-block">
        <ProSearchForm
          :form="deckSearchForm"
          :columns="deckSearchColumns"
          size="small"
          label-placement="left"
          label-width="84"
          cols="1 s:2 l:3"
          :collapse-button-props="false"
        />
      </section>

      <section class="deck-action-block">
        <n-flex size="small" align="center" class="deck-tools">
          <n-button
            type="info"
            size="tiny"
            text
            tag="a"
            target="_blank"
            href="https://github.com/sealdice/draw"
          >
            <template #icon>
              <n-icon><i-carbon-link /></n-icon>
            </template>
            获取牌堆
          </n-button>

          <input
            ref="fileInputRef"
            type="file"
            class="deck-file-input"
            multiple
            @change="onFileSelection"
          />
          <n-button type="info" secondary :loading="uploader.busy.value" @click="openFilePicker">
            <template #icon>
              <n-icon><i-carbon-upload /></n-icon>
            </template>
            上传牌堆
          </n-button>
        </n-flex>
      </section>

      <section v-if="activeUploadTasks.length" class="upload-panel">
        <div class="upload-panel-head">
          <h3>上传队列</h3>
          <n-tag size="small" :bordered="false" type="info">
            {{ activeUploadTasks.length }} 项
          </n-tag>
        </div>

        <div class="upload-list">
          <article v-for="task in activeUploadTasks" :key="task.id" class="upload-item">
            <div class="upload-item-head">
              <div class="upload-title">
                <span class="upload-name">{{ task.filename }}</span>
                <span class="upload-meta">{{ Math.round(task.fileSize / 1024) }} KB</span>
              </div>
              <div class="upload-actions">
                <n-tag
                  size="small"
                  :bordered="false"
                  :type="task.status === 'error' ? 'error' : task.status === 'success' ? 'success' : 'warning'"
                >
                  {{ task.status }}
                </n-tag>
                <n-button v-if="task.status === 'error'" size="tiny" secondary @click="retryTask(task)">
                  重试
                </n-button>
              </div>
            </div>

            <n-progress
              type="line"
              :percentage="task.progress"
              :status="task.status === 'error' ? 'error' : task.status === 'success' ? 'success' : 'default'"
              :show-indicator="true"
            />

            <div class="upload-detail">
              <span>分块 {{ Array.isArray(task.uploadedChunks) ? task.uploadedChunks.length : 0 }} / {{ task.expectedChunks || '-' }}</span>
              <span v-if="task.errorText" class="upload-error">{{ task.errorText }}</span>
            </div>
          </article>
        </div>
      </section>

      <aside class="deck-meta">
        <n-text v-if="filterCount > 0" class="text-xs" type="info">
          已过滤 {{ filterCount }} 条
        </n-text>

        <n-flex size="small" align="center" class="deck-meta-right">
          <n-text class="text-xs">目前支持 json/yaml/deck/toml 格式的牌堆</n-text>
          <n-tooltip>
            <template #trigger>
              <n-icon size="small"><i-carbon-help-filled /></n-icon>
            </template>
            deck 牌堆：一种单文件带图的牌堆格式<br />
            在牌堆文件中使用./images/xxx.png 的相对路径引用图片。并连同图片目录一起打包成 zip，修改扩展名为 deck 即可制作<br />
            <br />
            toml 牌堆：海豹支持的新牌堆格式。格式更加友好，还提供了包括云牌组在内的更多功能支持。
          </n-tooltip>
        </n-flex>
      </aside>

      <main class="deck-data-block">
        <FoldableCard
          v-for="(item, index) in items"
          :key="item.filename || index"
          class="deck-item"
          :err-title="item.filename"
          :err-text="item.errText"
        >
          <template #title>
            <n-flex size="small" align="center">
              <n-text class="text-base" tag="b">{{ item.name }}</n-text>
              <n-text>{{ item.version }}</n-text>
              <n-tag
                size="small"
                :type="item.fileFormat === 'toml' ? 'success' : 'info'"
                :bordered="false"
              >
                {{ item.fileFormat }}
              </n-tag>
            </n-flex>
          </template>

          <template #title-extra>
            <n-flex>
              <n-popconfirm
                v-if="item.updateUrls && item.updateUrls.length > 0"
                @positive-click="doCheckUpdate(item)"
              >
                <template #trigger>
                  <n-button type="info" size="small" secondary :loading="diffLoading">
                    <template #icon>
                      <n-icon><i-carbon-download /></n-icon>
                    </template>
                    更新
                  </n-button>
                </template>
                更新地址由牌堆作者提供，是否确认要检查该牌堆更新？
              </n-popconfirm>
              <n-button type="error" size="small" secondary @click="doDelete(item)">
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                删除
              </n-button>
            </n-flex>
          </template>

          <template #title-extra-error>
            <n-button type="error" size="small" secondary @click="doDelete(item)">
              <template #icon>
                <n-icon><i-carbon-row-delete /></n-icon>
              </template>
              删除
            </n-button>
          </template>

          <template #description>
            <n-flex size="small" vertical align="normal">
              <n-text v-if="item.cloud" type="info" class="text-xs">
                <n-icon><i-carbon-cloud /></n-icon>
                作者提供云端内容，请自行鉴别安全性
              </n-text>
              <n-text v-if="item.fileFormat === 'jsonc'" type="warning" class="text-xs">
                <n-icon><i-carbon-warning-filled /></n-icon>
                注意：该牌堆的格式并非标准 JSON，而是允许尾逗号与注释语法的扩展 JSON
              </n-text>
            </n-flex>
          </template>

          <n-descriptions content-class="whitespace-pre-line" :column="isMobile ? 1 : 3">
            <n-descriptions-item :span="3" label="作者">
              {{ item.author || '<佚名>' }}
            </n-descriptions-item>
            <n-descriptions-item v-if="item.desc" :span="3" label="简介">
              {{ item.desc }}
            </n-descriptions-item>
            <n-descriptions-item :span="3" label="牌组列表">
              <n-flex size="small">
                <n-tag
                  v-for="(visible, command) of item.command"
                  :key="command"
                  size="small"
                  :type="visible ? 'info' : 'default'"
                  :bordered="false"
                >
                  {{ command }}
                </n-tag>
              </n-flex>
            </n-descriptions-item>
            <n-descriptions-item v-if="item.license" label="许可协议">
              {{ item.license }}
            </n-descriptions-item>
            <n-descriptions-item v-if="item.date" label="发布时间">
              {{ item.date }}
            </n-descriptions-item>
            <n-descriptions-item v-if="item.updateDate" label="更新时间">
              {{ item.updateDate }}
            </n-descriptions-item>
          </n-descriptions>

          <template #unfolded-extra>
            <n-descriptions content-class="whitespace-pre-line" :column="isMobile ? 1 : 3">
              <n-descriptions-item :span="3" label="可见牌组列表">
                <n-flex size="small">
                  <n-tag
                    v-for="(visible, command) of item.command"
                    :key="command"
                    size="small"
                    :type="visible ? 'info' : 'default'"
                    :bordered="false"
                    :style="{ display: visible ? '' : 'none' }"
                  >
                    {{ command }}
                  </n-tag>
                </n-flex>
              </n-descriptions-item>
            </n-descriptions>
          </template>
        </FoldableCard>

        <n-empty v-if="!items.length" description="暂无牌堆" class="deck-empty" />
      </main>

      <div class="deck-pagination-block">
        <n-pagination
          v-model:page="listQuery.page"
          :page-size="listQuery.pageSize"
          :item-count="Number(total)"
          simple
        />
      </div>

      <n-modal v-model:show="showDiff" preset="card" title="牌堆内容对比" class="diff-dialog">
        <DiffViewer :lang="deckCheck.format ?? 'text'" :old="deckCheck.old ?? ''" :new="deckCheck.new ?? ''" />
        <template #footer>
          <n-flex wrap>
            <n-button @click="showDiff = false">取消</n-button>
            <n-button
              v-if="deckCheck.old !== deckCheck.new"
              type="success"
              :loading="updateMutation.isPending.value"
              @click="deckUpdate"
            >
              <template #icon>
                <n-icon><i-carbon-save /></n-icon>
              </template>
              确认更新
            </n-button>
          </n-flex>
        </template>
      </n-modal>
    </n-spin>
  </main>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, reactive, ref, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useDialog, useMessage } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import {
  getSdApiV2DeckList,
  postSdApiV2DeckCheckUpdate,
  postSdApiV2DeckDelete,
  postSdApiV2DeckReload,
  postSdApiV2DeckUpdate,
  postSdApiV2DeckUploadComplete,
  postSdApiV2DeckUploadInit,
  type DeckItem,
  type UpdateCheckResult,
} from '@/api';
import FoldableCard from '@/components/shared/FoldableCard.vue';
import { getApiBaseUrl } from '@/api/config';
import { useResumableUpload, type ResumableUploadTask } from '@/features/upload/resumableUpload';
import { hasAccessToken } from '@/features/auth/state';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';

const DiffViewer = defineAsyncComponent(() => import('@/components/shared/DiffViewer.vue'));

const deckChunkSize = 4 * 1024 * 1024;

// 牌堆页同时承载列表管理、diff 更新和大文件上传。
// 列表数据用 Vue Query；更新检查按需打开 DiffViewer；上传复用通用断点续传控制器。
const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  sortBy: 'name',
  sortOrder: 'asc',
});

type DeckSearchFormValues = {
  keyword: string;
  sortBy: string;
  sortOrder: string;
};

const defaultDeckSearchFormValues = (): DeckSearchFormValues => ({
  keyword: '',
  sortBy: 'name',
  sortOrder: 'asc',
});

const sortByOptions = [
  { label: '按名称', value: 'name' },
  { label: '按作者', value: 'author' },
  { label: '按更新时间', value: 'updateDate' },
];

const sortOrderOptions = [
  { label: '升序', value: 'asc' },
  { label: '降序', value: 'desc' },
];

const deckSearchForm = createProSearchForm<DeckSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultDeckSearchFormValues()),
  onSubmit: async values => {
    Object.assign(listQuery, {
      keyword: values.keyword,
      sortBy: values.sortBy,
      sortOrder: values.sortOrder,
      page: 1,
    });
    await deckListQuery.refetch();
  },
  onReset: async () => {
    Object.assign(listQuery, {
      ...defaultDeckSearchFormValues(),
      page: 1,
    });
    await deckListQuery.refetch();
  },
});

const deckSearchColumns: ProSearchFormColumns<DeckSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索牌堆 / 作者 / 简介 / 牌组',
    },
  },
  {
    label: '排序字段',
    path: 'sortBy',
    field: 'select',
    fieldProps: {
      options: sortByOptions,
    },
  },
  {
    label: '排序方向',
    path: 'sortOrder',
    field: 'select',
    fieldProps: {
      options: sortOrderOptions,
    },
  },
];

const showDiff = ref(false);
const diffLoading = ref(false);
const fileInputRef = ref<HTMLInputElement | null>(null);
const deckCheck = ref<UpdateCheckResult>({
  success: false,
  old: '',
  new: '',
  format: 'json',
  filename: '',
  tempFileName: '',
});

const deckListParams = computed(() => ({
  page: listQuery.page,
  pageSize: listQuery.pageSize,
  keyword: listQuery.keyword || undefined,
  sortBy: listQuery.sortBy,
  sortOrder: listQuery.sortOrder,
}));

const deckListQuery = useQuery({
  queryKey: computed(() => ['deck-list', deckListParams.value]),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2DeckList({
      query: deckListParams.value,
      throwOnError: true,
    });
    return data.item;
  },
});

const items = computed(() => deckListQuery.data.value?.list ?? []);
const total = computed(() => deckListQuery.data.value?.total ?? 0);
const filterCount = computed(() => {
  const count = Number(total.value) - items.value.length;
  return count > 0 ? count : 0;
});
const pageBusy = computed(() => deckListQuery.isFetching.value);

watch(
  () => listQuery.keyword,
  () => {
    listQuery.page = 1;
  },
);

watch(
  () => [listQuery.sortBy, listQuery.sortOrder] as const,
  () => {
    listQuery.page = 1;
  },
);

const invalidateDeckList = () =>
  queryClient.invalidateQueries({
    queryKey: ['deck-list'],
  });

const reloadMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2DeckReload({
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async item => {
    if (item.testMode) {
      message.success('展示模式无法重载牌堆');
      return;
    }
    message.success('已重载');
    await invalidateDeckList();
  },
  onError: () => {
    message.error('重载失败');
  },
});

const deleteMutation = useMutation({
  mutationFn: async (filename: string) => {
    const { data } = await postSdApiV2DeckDelete({
      body: {
        filename,
      },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async item => {
    if (!item.success) {
      message.error('删除失败');
      return;
    }
    message.success('牌堆已删除');
    await invalidateDeckList();
  },
  onError: () => {
    message.error('删除失败');
  },
});

const updateMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2DeckUpdate({
      body: {
        filename: deckCheck.value.filename ?? '',
        tempFileName: deckCheck.value.tempFileName ?? '',
      },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async item => {
    showDiff.value = false;
    if (!item.success) {
      message.error('更新失败');
      return;
    }
    message.success('更新成功，即将自动重载牌堆');
    await invalidateDeckList();
  },
  onError: () => {
    showDiff.value = false;
    message.error('更新失败');
  },
});

const uploader = useResumableUpload('sd-deck-upload-state', {
  chunkSize: deckChunkSize,
  async init(task: ResumableUploadTask) {
    const { data } = await postSdApiV2DeckUploadInit({
      body: {
        filename: task.filename,
        fileSize: task.fileSize,
        fileHash: task.fileHash,
        chunkSize: deckChunkSize,
      },
      throwOnError: true,
    });
    return {
      sessionId: data.item.sessionId,
      chunkSize: data.item.chunkSize,
      uploadedChunks: data.item.uploadedChunks ?? [],
      uploadedBytes: data.item.uploadedBytes,
      expectedChunks: data.item.expectedChunks,
    };
  },
  async complete(task: ResumableUploadTask): Promise<boolean> {
    const { data } = await postSdApiV2DeckUploadComplete({
      body: {
        sessionId: task.sessionId,
      },
      throwOnError: true,
    });
    return data.item.success;
  },
  buildChunkUrl(task: ResumableUploadTask, index: number): string {
    return `${getApiBaseUrl()}/sd-api/v2/deck/upload/${encodeURIComponent(task.sessionId)}/${index}`;
  },
  async onTaskSuccess(task: ResumableUploadTask) {
    message.success(`上传完成：${task.filename}`);
    await invalidateDeckList();
  },
  async onTaskError(task: ResumableUploadTask) {
    message.error(`上传失败：${task.filename}`);
  },
});

const activeUploadTasks = computed(() =>
  uploader.tasks.value.filter(task => task.status !== 'success'),
);

onMounted(() => {
  uploader.restore();
});

function openFilePicker() {
  fileInputRef.value?.click();
}

function onFileSelection(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  void uploader.enqueueFiles(files);
  input.value = '';
}

function doReload() {
  void reloadMutation.mutateAsync();
}

function retryTask(task: ResumableUploadTask) {
  void uploader.retry(task);
}

function doDelete(item: DeckItem) {
  dialog.warning({
    title: '确认删除',
    content: `确认删除牌堆「${item.name}」，确定吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(item.filename);
    },
  });
}

async function doCheckUpdate(item: DeckItem) {
  diffLoading.value = true;
  try {
    const { data } = await postSdApiV2DeckCheckUpdate({
      body: {
        filename: item.filename,
      },
      throwOnError: true,
    });
    if (!data.item.success) {
      message.error(`检查更新失败！${data.item.err ?? ''}`);
      return;
    }
    deckCheck.value = data.item;
    showDiff.value = true;
  } catch {
    message.error('检查更新失败');
  } finally {
    diffLoading.value = false;
  }
}

function deckUpdate() {
  void updateMutation.mutateAsync();
}
</script>

<style scoped>
.deck-page {
  width: 100%;
}

.page-header {
  margin-bottom: 1rem;
}

.deck-search-block {
  margin-bottom: 0.75rem;
}

.deck-action-block {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.deck-tools {
  margin-left: auto;
}

.deck-file-input {
  display: none;
}

.upload-panel {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.85rem;
  margin-bottom: 1rem;
}

.upload-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.upload-panel-head h3 {
  margin: 0;
  font-size: 0.95rem;
}

.upload-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.upload-item {
  border: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated-soft);
  padding: 0.75rem;
}

@supports (color: color-mix(in srgb, white, black)) {
  .upload-item {
    background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 20%);
  }
}

.upload-item-head,
.upload-title,
.upload-actions,
.upload-detail {
  display: flex;
  min-width: 0;
}

.upload-item-head,
.upload-detail {
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.upload-title {
  flex: 1 1 auto;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.upload-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upload-meta,
.upload-detail {
  color: var(--sd-text-muted);
  font-size: 0.8rem;
}

.upload-error {
  color: var(--n-error-color);
}

.deck-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.deck-meta-right {
  margin-left: auto;
}

.deck-data-block {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.deck-item {
  width: 100%;
}

.deck-pagination-block {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.deck-empty {
  padding: 2rem 0;
}

.diff-dialog {
  width: min(1000px, calc(100vw - 2rem));
}

@media screen and (max-width: 700px) {
  .deck-meta,
  .deck-action-block,
  .upload-item-head,
  .upload-detail {
    align-items: flex-start;
    flex-direction: column;
  }

  .deck-tools,
  .deck-meta-right {
    margin-left: 0;
  }

  .deck-pagination-block {
    justify-content: flex-start;
  }
}
</style>
