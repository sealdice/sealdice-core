<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { NButton, NFlex, NText, useDialog, useMessage } from 'naive-ui';
import {
  getSdApiV2StoryCleanupPreview,
  getSdApiV2StoryInfoOptions,
  getSdApiV2StoryItemsPage,
  getSdApiV2StoryLogsPage,
  postSdApiV2StoryCleanup,
  postSdApiV2StoryUploadLog,
  deleteSdApiV2StoryLog,
  type LogOneItem,
  type StoryLogView,
} from '@/api';
import StoryBackup from '@/components/story/StoryBackup.vue';
import FoldableCard from '@/components/shared/FoldableCard.vue';
import { hasAccessToken } from '@/features/auth/state';

dayjs.extend(relativeTime);

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

type StoryTab = 'list' | 'cleanup' | 'backup';
type StoryMode = 'logs' | 'items';

type LogView = StoryLogView & {
  pitch?: boolean;
};

const tab = ref<StoryTab>('list');
const mode = ref<StoryMode>('logs');

const queryLogPage = ref({
  pageNum: 1,
  pageSize: 20,
  total: 0,
  name: '',
  groupId: '',
  createdTime: null as unknown as [number, number],
});

const logItemPage = ref({
  pageNum: 1,
  pageSize: 100,
  size: 0,
  logId: 0,
});

const cleanupForm = ref({
  months: 6,
  vacuum: false,
});

const cleanupPreview = ref<{
  logs: number;
  items: number;
  oldestUpdated?: number;
  newestUpdated?: number;
  canVacuum: boolean;
} | null>(null);

const logs = ref<LogView[]>([]);
const itemData = ref<LogOneItem[]>([]);
const users = ref<Record<string, [string, string]>>({});

const storyInfoQuery = useQuery({
  ...getSdApiV2StoryInfoOptions(),
  enabled: hasAccessToken,
});

const summary = computed(() => storyInfoQuery.data.value?.item);

function linkStateText(log: LogView): string {
  switch (log.linkState) {
    case 'fresh':
      return '已有最新链接';
    case 'stale':
      return '链接已过期，建议重传';
    default:
      return '无链接';
  }
}

function linkStateType(log: LogView): 'default' | 'success' | 'warning' {
  switch (log.linkState) {
    case 'fresh':
      return 'success';
    case 'stale':
      return 'warning';
    default:
      return 'default';
  }
}

async function searchLogs() {
  const params = {
    pageNum: queryLogPage.value.pageNum,
    pageSize: queryLogPage.value.pageSize,
    name: queryLogPage.value.name || undefined,
    groupId: queryLogPage.value.groupId || undefined,
    createdTimeBegin: queryLogPage.value.createdTime?.[0]
      ? dayjs(queryLogPage.value.createdTime[0]).startOf('date').unix()
      : undefined,
    createdTimeEnd: queryLogPage.value.createdTime?.[1]
      ? dayjs(queryLogPage.value.createdTime[1]).endOf('date').unix()
      : undefined,
  };
  const { data } = await getSdApiV2StoryLogsPage({
    query: params,
    throwOnError: true,
  });
  if (data.item.result) {
    logs.value = (data.item.data ?? []).map(item => ({
      ...item,
      pitch: false,
    }));
    queryLogPage.value.total = data.item.total;
  } else {
    message.error('无法获取跑团日志' + (data.item.err ?? ''));
  }
}

const refreshLogs = async () => {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ['getSdApiV2StoryInfo'] }),
    searchLogs(),
  ]);
};

const deleteLogMutation = useMutation({
  mutationFn: async (log: LogView) => {
    const { data } = await deleteSdApiV2StoryLog({
      body: {
        body: {
          id: log.id,
        },
      },
      throwOnError: true,
    });
    return data.item;
  },
});

const uploadLogMutation = useMutation({
  mutationFn: async ({ log, force }: { log: LogView; force: boolean }) => {
    const { data } = await postSdApiV2StoryUploadLog({
      body: {
        body: {
          id: log.id,
          force,
        },
      },
      throwOnError: true,
    });
    return data.item;
  },
});

const cleanupMutation = useMutation({
  mutationFn: async (payload: { months: number; vacuum: boolean }) => {
    const { data } = await postSdApiV2StoryCleanup({
      body: {
        body: payload,
      },
      throwOnError: true,
    });
    return data.item;
  },
});

function handleLogPageChange(value: number) {
  queryLogPage.value.pageNum = value;
  void searchLogs();
}

function handlePageSizeChange(value: number) {
  queryLogPage.value.pageNum = 1;
  queryLogPage.value.pageSize = value;
  void searchLogs();
}

function delLog(log: LogView, refresh = true) {
  dialog.warning({
    title: '删除',
    content: '是否删除此跑团日志？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      const result = await deleteLogMutation.mutateAsync(log);
      if (result.success) {
        message.success('删除成功');
        if (refresh) {
          await refreshLogs();
        }
      } else {
        message.error('删除失败');
      }
    },
  });
}

function delLogs() {
  dialog.warning({
    title: '删除',
    content: '是否删除所选跑团日志？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      const selected = logs.value.filter(item => item.pitch);
      for (const log of selected) {
        const result = await deleteLogMutation.mutateAsync(log);
        if (result.success) {
          message.success('删除成功');
        } else {
          message.error('删除失败');
        }
      }
      await refreshLogs();
    },
  });
}

function showUploadResult(log: LogView, result: { url: string; reused: boolean; forced: boolean; unofficial: boolean }) {
  message.success(() => (
    <NFlex vertical>
      <NText>{result.reused ? '复用已有日志链接' : result.forced ? '强制重新上传成功' : '日志上传成功'}</NText>
      <NButton text type="primary" onClick={() => openLink(result.url)}>
        打开链接
      </NButton>
      {result.unofficial ? <NText depth={3}>注意：该链接非海豹官方染色器</NText> : null}
    </NFlex>
  ));
  log.uploadUrl = result.url;
  log.uploadTime = dayjs().unix();
  log.linkState = 'fresh';
}

function uploadLog(log: LogView, force = false) {
  dialog.warning({
    title: force ? '强制上传' : '上传日志',
    content: force ? '将重新上传此跑团日志并覆盖当前链接记录，是否继续？' : '将此跑团日志上传至海豹服务器？',
    positiveText: '确定',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      const result = await uploadLogMutation.mutateAsync({ log, force });
      showUploadResult(log, result);
    },
  });
}

async function openItem(log: LogView) {
  logItemPage.value.logId = log.id;
  logItemPage.value.size = log.size ?? 0;
  logItemPage.value.pageNum = 1;
  const { data } = await getSdApiV2StoryItemsPage({
    query: {
      logId: logItemPage.value.logId,
      pageNum: logItemPage.value.pageNum,
      pageSize: logItemPage.value.pageSize,
    },
    throwOnError: true,
  });
  itemData.value = data.item ?? [];
  mode.value = 'items';
}

async function handleItemPageChange(value: number) {
  logItemPage.value.pageNum = value;
  const { data } = await getSdApiV2StoryItemsPage({
    query: {
      logId: logItemPage.value.logId,
      pageNum: logItemPage.value.pageNum,
      pageSize: logItemPage.value.pageSize,
    },
    throwOnError: true,
  });
  itemData.value = data.item ?? [];
}

function closeItem() {
  itemData.value = [];
  mode.value = 'logs';
  users.value = {};
}

async function openCleanupDialog() {
  const months = Math.max(0, Math.trunc(cleanupForm.value.months || 0));
  const { data } = await getSdApiV2StoryCleanupPreview({
    query: { months },
    throwOnError: true,
  });
  cleanupPreview.value = data.item;
  const preview = cleanupPreview.value;
  dialog.warning({
    title: '日志清理',
    content: () => (
      <NFlex vertical>
        <NText>将删除 {preview.logs} 份超过 {months} 个月未更新的日志，共 {preview.items} 条消息。</NText>
        {cleanupForm.value.vacuum ? (
          <NText type="warning">这将可能导致海豹记录log用户运行缓慢，请注意</NText>
        ) : null}
      </NFlex>
    ),
    positiveText: '执行清理',
    negativeText: '取消',
    onPositiveClick: async () => {
      const result = await cleanupMutation.mutateAsync({
        months,
        vacuum: cleanupForm.value.vacuum,
      });
      message.success(`已删除 ${result.logs} 份日志、${result.items} 条消息${result.vacuumed ? '，并执行 VACUUM' : ''}`);
      await refreshLogs();
    },
  });
}

async function refreshCleanupPreview() {
  const months = Math.max(0, Math.trunc(cleanupForm.value.months || 0));
  const { data } = await getSdApiV2StoryCleanupPreview({
    query: { months },
    throwOnError: true,
  });
  cleanupPreview.value = data.item;
}

async function executeCleanup() {
  const months = Math.max(0, Math.trunc(cleanupForm.value.months || 0));
  const result = await cleanupMutation.mutateAsync({
    months,
    vacuum: cleanupForm.value.vacuum,
  });
  message.success(`已删除 ${result.logs} 份日志、${result.items} 条消息${result.vacuumed ? '，并执行 VACUUM' : ''}`);
  await Promise.all([refreshLogs(), refreshCleanupPreview()]);
}

function openLink(url: string) {
  if (!url) return;
  window.open(url, '_blank', 'noopener,noreferrer');
}

function randomColorWithIndex(index: number): string {
  const presets = [
    'var(--color-red-600)',
    'var(--color-orange-600)',
    'var(--color-yellow-600)',
    'var(--color-green-600)',
    'var(--color-cyan-600)',
    'var(--color-blue-600)',
    'var(--color-purple-600)',
    'var(--color-pink-600)',
    'var(--color-slate-600)',
  ];
  const randomColorSystems = ['red', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink', 'monochrome'];
  if (index < presets.length) {
    return presets[index];
  }
  return presets[index % presets.length];
}

const itemsView = computed(() => {
  const values: LogOneItem[] = [];
  itemData.value.forEach((item, index) => {
    if (!users.value[item.IMUserId]) {
      users.value[item.IMUserId] = [randomColorWithIndex(index), item.nickname];
    }
    values.push(item);
  });
  return values;
});

onMounted(async () => {
  await Promise.all([refreshLogs(), refreshCleanupPreview()]);
});
</script>

<template>
  <main class="story-page">
    <n-tabs v-model:value="tab" pane-class="mb-8" justify-content="space-evenly">
      <n-tab-pane tab="跑团日志" name="list">
        <template v-if="mode === 'logs'">
          <header class="page-header">
            <n-card title="跑团日志 / Story" :bordered="false">
              <n-flex vertical align="flex-start">
                <n-text>记录过 {{ summary?.totalLogs ?? 0 }} 份日志，共计 {{ summary?.totalItems ?? 0 }} 条消息</n-text>
                <n-text>现有 {{ summary?.currentLogs ?? 0 }} 份日志，共计 {{ summary?.currentItems ?? 0 }} 条消息</n-text>
              </n-flex>
            </n-card>
          </header>
          <section class="story-search-block">
            <div class="story-toolbar">
              <n-input v-model:value="queryLogPage.name" size="small" clearable placeholder="搜索日志名" />
              <n-input v-model:value="queryLogPage.groupId" size="small" clearable placeholder="搜索群号" />
              <n-date-picker v-model:value="queryLogPage.createdTime" size="small" type="daterange" clearable />
            </div>
            <n-flex size="small" align="center" class="story-tools">
              <n-button type="info" secondary @click="searchLogs">查询</n-button>
              <n-button
                secondary
                @click="
                  queryLogPage.name = '';
                  queryLogPage.groupId = '';
                  queryLogPage.createdTime = null as unknown as [number, number];
                  queryLogPage.pageNum = 1;
                  searchLogs();
                "
              >
                重置
              </n-button>
            </n-flex>
          </section>

          <section class="story-action-block">
            <n-flex size="small" align="center" class="story-tools">
              <n-button type="primary" size="small" @click="logs.forEach(item => (item.pitch = !item.pitch))">
                <template #icon>
                  <n-icon><i-carbon-checkmark /></n-icon>
                </template>
                全选
              </n-button>
              <n-button
                v-show="(logs?.filter(item => item.pitch)?.length ?? 0) > 0"
                type="error"
                size="small"
                @click="delLogs"
              >
                <template #icon>
                  <n-icon><i-carbon-row-delete /></n-icon>
                </template>
                删除所选
              </n-button>
            </n-flex>
          </section>

          <section class="story-data-block">
            <template v-for="log in logs" :key="log.id">
              <FoldableCard class="story-log-card">
                <template #title>
                  <n-flex align="center">
                    <n-checkbox v-model:checked="log.pitch" />
                    <n-flex align="center" wrap>
                      <n-text class="text-base" tag="strong">{{ log.name }}</n-text>
                      <n-text>({{ log.groupId }})</n-text>
                    </n-flex>
                  </n-flex>
                </template>

                <template #action>
                  <n-flex size="small">
                    <n-button size="small" secondary @click="openItem(log)">查看</n-button>
                    <n-button size="small" type="primary" secondary @click="uploadLog(log)">
                      <template #icon>
                        <n-icon><i-carbon-upload /></n-icon>
                      </template>
                      提取日志
                    </n-button>
                    <n-button size="small" secondary @click="uploadLog(log, true)">强制上传</n-button>
                    <n-button size="small" secondary :disabled="!log.uploadUrl" @click="openLink(log.uploadUrl)">
                      查看链接
                    </n-button>
                    <n-button size="small" type="error" secondary @click="delLog(log)">
                      <template #icon>
                        <n-icon><i-carbon-row-delete /></n-icon>
                      </template>
                      删除
                    </n-button>
                  </n-flex>
                </template>

                <n-flex vertical align="flex-start">
                  <n-flex>
                    <n-text>包含 {{ log.size ?? 0 }} 条消息</n-text>
                  </n-flex>
                  <n-flex align="center">
                    <n-text>链接状态：{{ linkStateText(log) }}</n-text>
                    <n-tag size="small" :type="linkStateType(log)" :bordered="false">
                      {{ log.linkState }}
                    </n-tag>
                  </n-flex>
                  <n-flex v-if="log.uploadTime">
                    <n-text>上传于：{{ dayjs.unix(log.uploadTime).format('YYYY-MM-DD HH:mm') }}</n-text>
                  </n-flex>
                  <n-flex>
                    <n-text>创建于：{{ dayjs.unix(log.createdAt).format('YYYY-MM-DD') }}</n-text>
                    <n-tag type="info" size="small" :bordered="false">
                      {{ dayjs.unix(log.createdAt).fromNow() }}
                    </n-tag>
                  </n-flex>
                  <n-flex>
                    <n-text>更新于：{{ dayjs.unix(log.updatedAt).format('YYYY-MM-DD') }}</n-text>
                    <n-tag type="info" size="small" :bordered="false">
                      {{ dayjs.unix(log.updatedAt).fromNow() }}
                    </n-tag>
                  </n-flex>
                </n-flex>
              </FoldableCard>
            </template>
          </section>

          <div class="story-pagination-block">
            <n-pagination
              v-model:page="queryLogPage.pageNum"
              v-model:page-size="queryLogPage.pageSize"
              show-size-picker
              :page-sizes="[10, 20, 30, 50]"
              :page-slot="3"
              :item-count="queryLogPage.total"
              @update:page="handleLogPageChange"
              @update:page-size="handlePageSizeChange"
            />
          </div>
        </template>

        <template v-else>
          <n-card title="跑团日志 / Story">
            <template #header-extra>
              <n-button type="primary" @click="closeItem">
                <template #icon>
                  <n-icon><i-carbon-chevron-left /></n-icon>
                </template>
                返回列表
              </n-button>
            </template>

            <n-collapse>
              <n-collapse-item title="颜色设置">
                <template v-for="(_, id) in users" :key="id">
                  <n-descriptions label-placement="top">
                    <n-descriptions-item :label="users[id][1]">
                      <n-color-picker
                        class="w-32"
                        v-model:value="users[id][0]"
                        :modes="['hex']"
                        :show-alpha="false"
                        :swatches="['#dc2626', '#ea580c', '#ca8a04', '#16a34a', '#0891b2', '#2563eb', '#9333ea', '#db2777']"
                      />
                    </n-descriptions-item>
                  </n-descriptions>
                </template>
              </n-collapse-item>
            </n-collapse>
          </n-card>

          <div class="story-item-list">
            <template v-for="(item, index) in itemsView" :key="index">
              <p :style="{ color: users[item.IMUserId][0] }">
                <span>{{ item.nickname }}：</span>
                <template v-for="(line, lineIndex) in item.message.split('\n')" :key="lineIndex">
                  <span>{{ line }}</span><br />
                </template>
              </p>
            </template>
          </div>

          <div class="story-pagination">
            <n-pagination
              v-model:page="logItemPage.pageNum"
              v-model:page-size="logItemPage.pageSize"
              show-size-picker
              :page-sizes="[50, 100, 200]"
              :page-slot="5"
              :item-count="logItemPage.size"
              @update:page="handleItemPageChange"
              @update:page-size="handleItemPageChange"
            />
          </div>
        </template>
      </n-tab-pane>

      <n-tab-pane tab="日志清理" name="cleanup">
        <section class="story-cleanup-page">
          <header class="page-header">
            <n-card title="日志清理" :bordered="false">
              <n-flex vertical align="flex-start">
                <n-text>按“超过 N 个月未更新”筛选日志并批量删除。</n-text>
                <n-text depth="3">清理只影响日志库，不影响 v1 接口。</n-text>
              </n-flex>
            </n-card>
          </header>

          <section class="cleanup-panel">
            <div class="cleanup-panel-head">
              <div>
                <h3>清理参数</h3>
                <p>先预览，再执行危险操作。</p>
              </div>
              <n-button secondary @click="refreshCleanupPreview">刷新预览</n-button>
            </div>

            <div class="cleanup-panel-body">
              <div class="cleanup-toolbar">
                <n-input-number v-model:value="cleanupForm.months" :min="0" class="cleanup-months" />
                <n-switch v-model:value="cleanupForm.vacuum" />
              </div>
              <div class="cleanup-toolbar-labels">
                <n-text depth="3">N 个月未更新</n-text>
                <n-text depth="3">执行 VACUUM</n-text>
              </div>
            </div>
          </section>

          <section class="cleanup-panel">
            <div class="cleanup-panel-head">
              <div>
                <h3>预览结果</h3>
                <p>依据当前阈值估算待删除范围。</p>
              </div>
            </div>

            <div class="cleanup-stats">
              <n-card size="small">
                <n-statistic label="待删日志" :value="cleanupPreview?.logs ?? 0" />
              </n-card>
              <n-card size="small">
                <n-statistic label="待删消息" :value="cleanupPreview?.items ?? 0" />
              </n-card>
              <n-card size="small">
                <n-statistic
                  label="最早更新时间"
                  :value="cleanupPreview?.oldestUpdated ? dayjs.unix(cleanupPreview.oldestUpdated).format('YYYY-MM-DD') : '--'"
                />
              </n-card>
              <n-card size="small">
                <n-statistic
                  label="最近更新时间"
                  :value="cleanupPreview?.newestUpdated ? dayjs.unix(cleanupPreview.newestUpdated).format('YYYY-MM-DD') : '--'"
                />
              </n-card>
            </div>
          </section>

          <section class="cleanup-panel cleanup-danger">
            <div class="cleanup-panel-head">
              <div>
                <h3>执行清理</h3>
                <p>危险操作，不可撤销。</p>
              </div>
            </div>

            <n-alert v-if="cleanupForm.vacuum" type="warning" :show-icon="false" class="cleanup-alert">
              这将可能导致海豹记录log用户运行缓慢，请注意
            </n-alert>

            <div class="cleanup-actions">
              <n-button :loading="cleanupMutation.isPending.value" type="error" @click="openCleanupDialog">
                确认并执行
              </n-button>
              <n-button secondary @click="executeCleanup" v-if="false">执行清理</n-button>
            </div>
          </section>
        </section>
      </n-tab-pane>

      <n-tab-pane tab="日志备份" name="backup">
        <StoryBackup />
      </n-tab-pane>
    </n-tabs>
  </main>
</template>

<style scoped>
.page-header {
  margin-bottom: 1rem;
}

.story-search-block {
  display: flex;
  flex-wrap: wrap-reverse;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.story-toolbar {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(180px, 1fr) minmax(260px, 320px);
  gap: 0.5rem;
  min-width: 0;
  flex: 1 1 640px;
}

.story-tools {
  margin-left: auto;
}

.story-action-block {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 1rem;
}

.story-data-block {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.story-log-card {
  width: 100%;
}

.story-item-list {
  margin: 1rem 0;
  padding: 0 1rem;
}

.story-pagination-block {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.story-cleanup-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cleanup-panel {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  padding: 0.95rem;
}

.cleanup-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.85rem;
}

.cleanup-panel-head h3 {
  margin: 0;
  font-size: 0.98rem;
}

.cleanup-panel-head p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}

.cleanup-panel-body {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.cleanup-toolbar,
.cleanup-toolbar-labels,
.cleanup-actions {
  display: grid;
  grid-template-columns: minmax(120px, 180px) 120px;
  gap: 0.75rem;
  align-items: center;
}

.cleanup-months {
  min-width: 0;
}

.cleanup-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.cleanup-danger {
  border-color: color-mix(in srgb, var(--n-error-color), var(--sd-border) 60%);
}

.cleanup-alert {
  margin-bottom: 0.85rem;
}

@media screen and (max-width: 900px) {
  .story-toolbar,
  .cleanup-stats {
    grid-template-columns: 1fr;
  }

  .cleanup-toolbar,
  .cleanup-toolbar-labels,
  .cleanup-actions {
    grid-template-columns: 1fr;
  }
}

@media screen and (max-width: 700px) {
  .story-search-block,
  .story-action-block {
    flex-direction: column;
    align-items: flex-start;
  }

  .story-tools {
    margin-left: 0;
  }
}
</style>
