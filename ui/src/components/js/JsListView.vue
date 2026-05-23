<script setup lang="tsx">
import { computed, defineAsyncComponent, onMounted, reactive, ref, watch } from 'vue';
import dayjs from 'dayjs';
import { NButton, NCheckbox, NFlex, NPagination, NTag, NText, useDialog, useMessage, type UploadCustomRequestOptions } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import { type JsInfo as JsInfoType } from '@/api';
import FoldableCard from '@/components/shared/FoldableCard.vue';
import { type ResumableUploadTask } from '@/features/upload/resumableUpload';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';
import { type JsUpdateDiff, useJsList } from '@/features/js/useJsList';

const DiffViewer = defineAsyncComponent(() => import('@/components/shared/DiffViewer.vue'));

const emit = defineEmits<{
  markNeedReload: [];
}>();

const message = useMessage();
const dialog = useDialog();

interface JsInfoExt extends JsInfoType {
  pitch?: boolean;
}

const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  sortBy: 'name',
  sortOrder: 'asc' as 'asc' | 'desc',
});

type JsListSearchFormValues = {
  keyword: string;
  sortBy: string;
  sortOrder: 'asc' | 'desc';
};

const sortByOptions = [
  { label: '按名称', value: 'name' },
  { label: '按作者', value: 'author' },
  { label: '按版本', value: 'version' },
  { label: '按安装时间', value: 'installTime' },
  { label: '按更新时间', value: 'updateTime' },
];

const sortOrderOptions = [
  { label: '升序', value: 'asc' },
  { label: '降序', value: 'desc' },
];

const defaultJsListSearchFormValues = (): JsListSearchFormValues => ({
  keyword: '',
  sortBy: 'name',
  sortOrder: 'asc',
});

const searchForm = createProSearchForm<JsListSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultJsListSearchFormValues()),
  onSubmit: async values => {
    Object.assign(listQuery, values, { page: 1 });
    await listQueryResult.refetch();
  },
  onReset: async () => {
    resetFilters();
  },
});

const searchColumns: ProSearchFormColumns<JsListSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索名称/描述/作者',
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

const listParams = computed(() => ({
  page: listQuery.page,
  pageSize: listQuery.pageSize,
  keyword: listQuery.keyword || undefined,
  sortBy: listQuery.sortBy,
  sortOrder: listQuery.sortOrder,
}));
const {
  listQueryResult,
  invalidateList,
  deleteMutation,
  enableMutation,
  disableMutation,
  uploader,
  checkUpdate,
  applyUpdate,
} = useJsList({
  listParams,
  async onUploadSuccess(task: ResumableUploadTask) {
    message.success(`上传完成：${task.filename}`);
    emit('markNeedReload');
    await invalidateList();
  },
  onUploadError(task: ResumableUploadTask) {
    message.error(`上传失败：${task.filename}`);
  },
});

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

const items = computed<JsInfoExt[]>(() =>
  (listQueryResult.data.value?.data ?? []).map(item => ({ ...item, pitch: false })),
);
const total = computed(() => listQueryResult.data.value?.total ?? 0);
const hasItems = computed(() => items.value.length > 0);
const selectedCount = computed(() => items.value.filter(item => item.pitch).length);
const allSelected = computed(() => hasItems.value && items.value.every(item => item.pitch));
const showPagination = computed(() => total.value > listQuery.pageSize);
const filterHint = computed(() => {
  if (!listQuery.keyword.trim()) return '';
  return `当前匹配 ${total.value} 条`;
});

const showDiff = ref(false);
const diffLoading = ref(false);
const diffData = ref<JsUpdateDiff | null>(null);

const activeUploadTasks = computed(() =>
  uploader.tasks.value.filter(task => task.status !== 'success'),
);

onMounted(() => {
  uploader.restore();
});

async function uploadPlugin(options: UploadCustomRequestOptions) {
  const file = options.file.file as File;
  if (!file) {
    options.onError();
    return;
  }
  const ext = file.name.split('.').pop()?.toLowerCase();
  if (ext !== 'js' && ext !== 'ts') {
    message.error('仅支持上传 .js 或 .ts 格式的插件文件');
    options.onError();
    return;
  }
  try {
    await uploader.enqueueFiles([file]);
    options.onFinish();
  } catch {
    message.error('上传失败');
    options.onError();
  }
}

async function handleCheckUpdate(item: JsInfoExt) {
  diffLoading.value = true;
  try {
    diffData.value = await checkUpdate(item.filename);
    showDiff.value = true;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '检查更新失败');
  } finally {
    diffLoading.value = false;
  }
}

async function handleApplyUpdate() {
  if (!diffData.value) return;
  try {
    await applyUpdate(diffData.value);
    message.success('更新成功，请手动重载后生效');
    showDiff.value = false;
    emit('markNeedReload');
    await invalidateList();
  } catch {
    message.error('更新失败');
  }
}

async function handleDelete(item: JsInfoExt) {
  dialog.warning({
    title: item.official ? '确认卸载' : '确认删除',
    content: item.official
      ? `确认卸载官方插件「${item.name}」的更新，确定吗？`
      : `确认删除插件「${item.name}」，确定吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteMutation.mutateAsync(item.filename);
        message.success('插件已删除，请手动重载后生效');
        emit('markNeedReload');
        await invalidateList();
      } catch {
        message.error('删除失败');
      }
    },
  });
}

async function toggleEnable(item: JsInfoExt, enable: boolean) {
  try {
    if (enable) {
      await enableMutation.mutateAsync(item.name);
      message.success('插件已启用，请手动重载后生效');
    } else {
      await disableMutation.mutateAsync(item.name);
      message.success('插件已禁用，请手动重载后生效');
    }
    emit('markNeedReload');
    await invalidateList();
  } catch {
    message.error('操作失败');
  }
}

async function delSelected() {
  const selected = items.value.filter(i => i.pitch);
  if (!selected.length) return;
  dialog.warning({
    title: '批量删除',
    content: `确认删除 ${selected.length} 个插件？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      for (const item of selected) {
        try {
          await deleteMutation.mutateAsync(item.filename);
        } catch {
          // continue
        }
      }
      message.success('删除完成，请手动重载后生效');
      emit('markNeedReload');
      await invalidateList();
    },
  });
}

function resetFilters() {
  listQuery.keyword = '';
  listQuery.sortBy = 'name';
  listQuery.sortOrder = 'asc';
  listQuery.page = 1;
  void listQueryResult.refetch();
}

function toggleSelectAll(checked: boolean) {
  items.value.forEach(item => {
    item.pitch = checked;
  });
}
</script>

<template>
  <main class="js-list-page">
    <section class="js-panel">
      <header class="js-panel-header">
        <div class="js-query-main">
          <ProSearchForm
            :form="searchForm"
            :columns="searchColumns"
            size="small"
            label-placement="left"
            label-width="72"
            cols="1 s:2 l:3"
            :collapse-button-props="false"
          />
          <div class="js-query-footer">
            <aside class="js-query-meta">
              <n-text v-if="filterHint" type="info" class="text-xs">
                {{ filterHint }}
              </n-text>
              <n-text v-else depth="3" class="text-xs">
                支持按插件名称、简介和作者筛选
              </n-text>
            </aside>

            <div class="js-query-tools">
              <n-upload
                action=""
                multiple
                accept="application/javascript,application/typescript,.js,.ts"
                :show-file-list="false"
                :custom-request="uploadPlugin"
              >
                <n-button type="info" secondary :loading="uploader.busy.value">
                  <template #icon>
                    <n-icon><i-carbon-upload /></n-icon>
                  </template>
                  上传插件
                </n-button>
              </n-upload>
              <n-button
                type="info"
                size="tiny"
                text
                tag="a"
                target="_blank"
                style="text-decoration: none"
                href="https://github.com/sealdice/javascript"
              >
                <template #icon>
                  <n-icon><i-carbon-link /></n-icon>
                </template>
                获取插件
              </n-button>
            </div>
          </div>
        </div>
      </header>
    </section>

    <section v-if="activeUploadTasks.length" class="js-panel upload-panel">
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
              <n-button v-if="task.status === 'error'" size="tiny" secondary @click="uploader.retry(task)">
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

    <section class="js-panel">
      <header class="js-panel-header js-data-header">
        <div class="js-batch-actions">
          <n-checkbox
            :checked="allSelected"
            :disabled="!hasItems"
            @update:checked="toggleSelectAll"
          >
            全选
          </n-checkbox>
          <n-text depth="3" class="text-xs">
            {{ hasItems ? `本页 ${items.length} 项，已选 ${selectedCount} 项` : '当前没有可操作的插件' }}
          </n-text>
        </div>
        <n-button
          type="error"
          size="small"
          secondary
          :disabled="selectedCount === 0"
          @click="delSelected"
        >
          <template #icon>
            <n-icon><i-carbon-row-delete /></n-icon>
          </template>
          删除所选
        </n-button>
      </header>

      <div class="js-panel-body">
        <section v-if="hasItems" class="js-list-main">
          <template v-for="item in items" :key="item.filename">
            <FoldableCard class="js-plugin-card" :err-title="item.filename" :err-text="item.errText">
          <template #title>
            <n-flex align="center">
              <n-checkbox v-model:checked="item.pitch" />
              <n-switch
                v-model:value="item.enable"
                :disabled="item.errText !== ''"
                @update:value="(v: boolean) => toggleEnable(item, v)"
              />
              <n-text tag="strong" class="text-base">{{ item.name }}</n-text>
              <n-text>{{ item.version || '<未定义>' }}</n-text>
              <n-tag v-if="item.official" size="small" type="primary" :bordered="false">官方</n-tag>
              <n-tag v-if="item.filename.toLowerCase().endsWith('.ts')" size="small" type="info" :bordered="false">TS</n-tag>
            </n-flex>
          </template>

          <template #title-extra>
            <n-flex>
              <n-button
                v-if="item.official && item.updateUrls && item.updateUrls.length > 0"
                type="info"
                size="small"
                secondary
                disabled
              >
                <template #icon>
                  <n-icon><i-carbon-download /></n-icon>
                </template>
                更新
              </n-button>
              <n-button
                v-else-if="item.updateUrls && item.updateUrls.length > 0"
                type="success"
                size="small"
                secondary
                :loading="diffLoading"
                @click="handleCheckUpdate(item)"
              >
                <template #icon>
                  <n-icon><i-carbon-download /></n-icon>
                </template>
                更新
              </n-button>

              <n-button
                v-if="item.builtin && item.builtinUpdated"
                type="error"
                size="small"
                secondary
                @click="handleDelete(item)"
              >
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                卸载更新
              </n-button>
              <n-button
                v-else-if="!item.builtin"
                type="error"
                size="small"
                secondary
                @click="handleDelete(item)"
              >
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                删除
              </n-button>
            </n-flex>
          </template>

          <template #title-extra-error>
            <n-flex>
              <n-button
                v-if="item.builtin && item.builtinUpdated"
                type="error"
                size="small"
                secondary
                @click="handleDelete(item)"
              >
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                卸载更新
              </n-button>
              <n-button
                v-else-if="!item.builtin"
                type="error"
                size="small"
                secondary
                @click="handleDelete(item)"
              >
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                删除
              </n-button>
            </n-flex>
          </template>

          <n-descriptions content-class="whitespace-pre-line">
            <n-descriptions-item v-if="!item.official" :span="3" label="作者">
              {{ item.author || '<佚名>' }}
            </n-descriptions-item>
            <n-descriptions-item :span="3" label="介绍">
              {{ item.desc || '<暂无>' }}
            </n-descriptions-item>
            <n-descriptions-item v-if="!item.official" :span="3" label="主页">
              {{ item.homepage || '<暂无>' }}
            </n-descriptions-item>
            <n-descriptions-item label="许可协议">
              {{ item.license || '<暂无>' }}
            </n-descriptions-item>
            <n-descriptions-item label="安装时间">
              {{ dayjs.unix(item.installTime).fromNow() }}
            </n-descriptions-item>
            <n-descriptions-item label="更新时间">
              {{ item.updateTime ? dayjs.unix(item.updateTime).fromNow() : '<暂无>' }}
            </n-descriptions-item>
          </n-descriptions>

          <template #unfolded-extra>
            <n-ellipsis :line-clamp="2">{{ item.desc || '<暂无>' }}</n-ellipsis>
          </template>
            </FoldableCard>
          </template>
        </section>

        <div v-else-if="!listQueryResult.isFetching.value" class="js-empty-state">
          <n-empty description="暂无插件">
            <template #icon>
              <n-icon size="40"><i-carbon-container-software /></n-icon>
            </template>
            <template #extra>
              <n-text depth="3">可以先上传插件，或到插件仓库获取脚本后再导入。</n-text>
            </template>
          </n-empty>
        </div>
      </div>
    </section>

    <section v-if="showPagination" class="js-panel js-pagination-panel">
      <div class="js-pagination-block">
        <n-pagination
          v-model:page="listQuery.page"
          v-model:page-size="listQuery.pageSize"
          show-size-picker
          :page-sizes="[10, 20, 30, 50]"
          :item-count="total"
          :page-slot="3"
        />
      </div>
    </section>

    <n-modal v-model:show="showDiff" title="插件内容对比" preset="card" style="width: 90vw; max-width: 1000px">
      <DiffViewer
        v-if="diffData"
        :old="diffData.old"
        :new="diffData.new"
        lang="javascript"
        class="max-h-150 overflow-auto"
      />
      <template #footer>
        <n-flex justify="end">
          <n-button @click="showDiff = false">取消</n-button>
          <n-button
            v-if="diffData && diffData.old !== diffData.new"
            type="primary"
            @click="handleApplyUpdate"
          >
            确认更新
          </n-button>
        </n-flex>
      </template>
    </n-modal>
  </main>
</template>

<style scoped>
.js-list-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.js-panel {
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: 16px;
  background:
    linear-gradient(180deg, var(--sd-bg-elevated-tint) 0%, var(--sd-bg-elevated) 14%, var(--sd-bg-elevated) 100%);
  box-shadow:
    0 14px 32px rgba(15, 23, 42, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.35);
}

.dark .js-panel {
  box-shadow:
    0 20px 44px rgba(2, 6, 23, 0.32),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.js-panel-header {
  padding: 0.95rem 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--sd-bg-elevated-soft), transparent 10%) 0%, transparent 100%);
}

.js-panel-body {
  padding: 1rem;
}

.js-query-main {
  display: grid;
  gap: 0.75rem;
  min-width: 0;
}

.js-query-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.js-query-tools {
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
}

.upload-panel {
  padding: 0.95rem 1rem 1rem;
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

.js-data-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.js-batch-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.js-list-main {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.js-plugin-card {
  width: 100%;
}

.js-empty-state {
  display: flex;
  min-height: 17rem;
  align-items: center;
  justify-content: center;
  padding: 1rem 0;
}

.js-pagination-panel {
  padding: 0.85rem 1rem;
}

.js-pagination-block {
  display: flex;
  justify-content: flex-start;
}

.max-h-150 {
  max-height: 37.5rem;
}

@supports (color: color-mix(in srgb, white, black)) {
  .upload-item {
    background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 20%);
  }
}

@media screen and (max-width: 700px) {
  .js-query-footer,
  .js-data-header,
  .upload-item-head,
  .upload-detail {
    align-items: flex-start;
    flex-direction: column;
  }

  .js-query-tools {
    justify-content: flex-start;
  }
}
</style>
