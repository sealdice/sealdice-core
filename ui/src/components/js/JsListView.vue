<script setup lang="tsx">
import { computed, defineAsyncComponent, onMounted, reactive, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { NButton, NCheckbox, NFlex, NInput, NPagination, NSelect, NTag, NText, useDialog, useMessage, type UploadCustomRequestOptions } from 'naive-ui';
import {
  getSdApiV2JsList,
  postSdApiV2JsDelete,
  postSdApiV2JsEnable,
  postSdApiV2JsDisable,
  postSdApiV2JsCheckUpdate,
  postSdApiV2JsUpdate,
  type JsInfo as JsInfoType,
} from '@/api';
import { getApiBaseUrl } from '@/api/config';
import {
  postSdApiV2JsUploadComplete,
  postSdApiV2JsUploadInit,
} from '@/api/generated';
import FoldableCard from '@/components/shared/FoldableCard.vue';
import { hasAccessToken } from '@/features/auth/state';
import { useResumableUpload, type ResumableUploadTask } from '@/features/upload/resumableUpload';

const DiffViewer = defineAsyncComponent(() => import('@/components/shared/DiffViewer.vue'));

dayjs.extend(relativeTime);

const emit = defineEmits<{
  markNeedReload: [];
}>();

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

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

const listParams = computed(() => ({
  page: listQuery.page,
  pageSize: listQuery.pageSize,
  keyword: listQuery.keyword || undefined,
  sortBy: listQuery.sortBy,
  sortOrder: listQuery.sortOrder,
}));

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

const listQueryResult = useQuery({
  queryKey: computed(() => ['js-list', listParams.value]),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2JsList({
      query: listParams.value,
      throwOnError: true,
    });
    return data.item;
  },
});

const items = computed<JsInfoExt[]>(() =>
  (listQueryResult.data.value?.data ?? []).map(item => ({ ...item, pitch: false })),
);
const total = computed(() => listQueryResult.data.value?.total ?? 0);
const filterHint = computed(() => {
  if (!listQuery.keyword.trim()) return '';
  return `当前匹配 ${total.value} 条`;
});

const invalidateList = () => queryClient.invalidateQueries({ queryKey: ['js-list'] });

const deleteMutation = useMutation({
  mutationFn: async (filename: string) => {
    await postSdApiV2JsDelete({
      body: { body: { filename } },
      throwOnError: true,
    });
  },
});

const enableMutation = useMutation({
  mutationFn: async (name: string) => {
    await postSdApiV2JsEnable({
      body: { body: { name } },
      throwOnError: true,
    });
  },
});

const disableMutation = useMutation({
  mutationFn: async (name: string) => {
    await postSdApiV2JsDisable({
      body: { body: { name } },
      throwOnError: true,
    });
  },
});

const showDiff = ref(false);
const diffLoading = ref(false);
const diffData = ref<{ old: string; new: string; filename: string; tempFileName: string } | null>(null);
const uploader = useResumableUpload('sd-js-upload-state', {
  chunkSize: 4 * 1024 * 1024,
  async init(task: ResumableUploadTask) {
    const { data } = await postSdApiV2JsUploadInit({
      body: {
        body: {
          filename: task.filename,
          fileSize: task.fileSize,
          fileHash: task.fileHash,
          chunkSize: 4 * 1024 * 1024,
        },
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
  async complete(task: ResumableUploadTask) {
    const { data } = await postSdApiV2JsUploadComplete({
      body: {
        body: {
          sessionId: task.sessionId,
        },
      },
      throwOnError: true,
    });
    return data.item.success;
  },
  buildChunkUrl(task: ResumableUploadTask, index: number) {
    return `${getApiBaseUrl()}/sd-api/v2/js/upload/${encodeURIComponent(task.sessionId)}/${index}`;
  },
  async onTaskSuccess(task: ResumableUploadTask) {
    message.success(`上传完成：${task.filename}`);
    emit('markNeedReload');
    await invalidateList();
  },
  onTaskError(task: ResumableUploadTask) {
    message.error(`上传失败：${task.filename}`);
  },
});

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
    const { data } = await postSdApiV2JsCheckUpdate({
      body: { body: { filename: item.filename } },
      throwOnError: true,
    });
    if (!data.item.success) {
      message.error(data.item.err || '检查更新失败');
      return;
    }
    diffData.value = {
      old: data.item.old || '',
      new: data.item.new || '',
      filename: data.item.filename || '',
      tempFileName: data.item.tempFileName || '',
    };
    showDiff.value = true;
  } catch {
    message.error('检查更新失败');
  } finally {
    diffLoading.value = false;
  }
}

async function handleApplyUpdate() {
  if (!diffData.value) return;
  try {
    await postSdApiV2JsUpdate({
      body: {
        body: {
          filename: diffData.value.filename,
          tempFileName: diffData.value.tempFileName,
        },
      },
      throwOnError: true,
    });
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
</script>

<template>
  <div>
    <section class="js-search-block">
      <div class="js-toolbar-left">
        <n-flex align="center" wrap size="small">
          <n-input v-model:value="listQuery.keyword" placeholder="搜索名称/描述/作者" clearable class="js-search" />
          <n-button type="info" secondary @click="listQueryResult.refetch()">查询</n-button>
          <n-button
            secondary
            @click="
              listQuery.keyword = '';
              listQuery.sortBy = 'name';
              listQuery.sortOrder = 'asc';
              listQuery.page = 1;
              listQueryResult.refetch();
            "
          >
            重置
          </n-button>
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
        </n-flex>
        <aside class="js-filter-hint">
          <n-text v-if="filterHint" type="info" class="text-xs">
            {{ filterHint }}
          </n-text>
        </aside>
      </div>
    </section>

    <section class="js-action-block">
      <div class="js-toolbar-right">
        <n-flex size="small" align="center" wrap>
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
          <n-select
            v-model:value="listQuery.sortBy"
            :options="sortByOptions"
            class="js-sort"
            size="small"
            placeholder="排序"
          />
          <n-select
            v-model:value="listQuery.sortOrder"
            :options="sortOrderOptions"
            class="js-order"
            size="small"
          />
        </n-flex>
      </div>
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

    <section class="js-batch-actions-block">
      <n-button-group class="js-batch-actions">
      <n-button size="small" @click="items.forEach(i => (i.pitch = !i.pitch))">
        <template #icon>
          <n-icon><i-carbon-checkmark /></n-icon>
        </template>
        全选
      </n-button>
      <n-button
        v-if="items.filter(i => i.pitch).length"
        type="error"
        size="small"
        @click="delSelected"
      >
        <template #icon>
          <n-icon><i-carbon-row-delete /></n-icon>
        </template>
        删除所选
      </n-button>
      </n-button-group>
    </section>

    <section class="js-list-main">
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

      <n-text v-if="!items.length && !listQueryResult.isFetching.value" depth="3" class="mt-4">
        暂无插件
      </n-text>
    </section>

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
  </div>
</template>

<style scoped>
.js-search-block {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.js-toolbar-left {
  min-width: 0;
  flex: 1 1 320px;
}

.js-toolbar-right {
  margin-left: auto;
}

.js-action-block {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 1rem;
}

.js-search {
  width: 15rem;
}

.js-sort {
  width: 8rem;
}

.js-order {
  width: 5rem;
}

.js-filter-hint {
  margin-top: 1rem;
}

.js-batch-actions-block {
  margin-bottom: 1rem;
}

.js-batch-actions {
  display: block;
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
  background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 20%);
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

.js-list-main {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-top: 1rem;
}

.js-plugin-card {
  width: 100%;
}

.js-pagination-block {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.max-h-150 {
  max-height: 37.5rem;
}

@media screen and (max-width: 700px) {
  .js-toolbar-right {
    margin-left: 0;
  }

  .upload-item-head,
  .upload-detail {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
