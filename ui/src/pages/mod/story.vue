<template>
  <main class="story-page sd-page-flow">
    <PageHeader title="跑团日志" />
    <n-tabs v-model:value="tab" class="story-tabs sd-scrollable-tabs">
      <n-tab-pane tab="跑团日志" name="list">
        <template v-if="mode === 'logs'">
          <ListWorkspace class="story-tab-body">
            <!-- 统计是数据读出，不是说明，用文本行承载即可，不套卡片。 -->
            <n-flex class="story-summary" size="small" wrap>
              <n-text depth="3">
                记录过 {{ summary?.totalLogs ?? 0 }} 份日志，共计
                {{ summary?.totalItems ?? 0 }} 条消息
              </n-text>
              <n-text depth="3">
                现有 {{ summary?.currentLogs ?? 0 }} 份日志，共计
                {{ summary?.currentItems ?? 0 }} 条消息
              </n-text>
            </n-flex>

            <QueryToolbar :form="storySearchForm" :columns="storySearchColumns" cols="1 s:2 l:3" />

            <ListPanel>
              <template #toolbar>
                <ResultToolbar>
                  <template #meta>
                    <n-text depth="3"
                      >共 {{ queryLogPage.total }} 项，本页 {{ logs.length }} 项</n-text
                    >
                    <n-checkbox
                      :checked="allLogsSelected"
                      aria-label="全选当前页日志"
                      @update:checked="toggleSelectAll"
                    >
                      {{ allLogsSelected ? '全不选' : '全选' }}
                    </n-checkbox>
                    <n-text depth="3" class="story-selected-count"
                      >已选 {{ selectedCount }} 项</n-text
                    >
                  </template>

                  <n-button
                    v-if="selectedCount > 0"
                    size="small"
                    type="primary"
                    :loading="uploadLogMutation.isPending.value"
                    @click="batchUploadLogs"
                  >
                    <template #icon>
                      <n-icon><i-tabler-upload /></n-icon>
                    </template>
                    批量提取日志
                  </n-button>
                  <n-button v-if="selectedCount > 0" size="small" type="error" @click="delLogs">
                    <template #icon>
                      <n-icon><i-tabler-trash /></n-icon>
                    </template>
                    删除所选
                  </n-button>
                </ResultToolbar>
              </template>

              <ListEmptyState v-if="!logs.length" description="暂无跑团日志" />

              <template v-for="log in logs" :key="log.id">
                <FoldableCard class="story-log-card">
                  <template #title>
                    <n-flex align="center">
                      <n-checkbox
                        v-model:checked="log.pitch"
                        :aria-label="`选择日志 ${log.name}`"
                      />
                      <n-flex align="center" wrap>
                        <n-text class="text-base" tag="strong">{{ log.name }}</n-text>
                        <n-text>({{ log.groupId }})</n-text>
                      </n-flex>
                    </n-flex>
                  </template>

                  <template #action>
                    <n-flex size="small" wrap>
                      <n-button text type="primary" @click="openItem(log)">查看</n-button>
                      <n-button size="small" type="primary" @click="uploadLog(log)">
                        <template #icon>
                          <n-icon><i-tabler-upload /></n-icon>
                        </template>
                        提取日志
                      </n-button>
                      <n-dropdown
                        trigger="click"
                        :options="logActionOptions(log)"
                        @select="(key: string) => handleLogAction(key, log)"
                      >
                        <n-button text aria-label="更多操作">
                          更多
                          <template #icon
                            ><n-icon><i-tabler-dots /></n-icon
                          ></template>
                        </n-button>
                      </n-dropdown>
                    </n-flex>
                  </template>

                  <n-flex vertical align="flex-start">
                    <n-flex>
                      <n-text>包含 {{ log.size ?? 0 }} 条消息</n-text>
                    </n-flex>
                    <n-flex align="center">
                      <n-text>链接状态：{{ linkStateText(log) }}</n-text>
                      <n-tag size="small" :type="linkStateType(log)" :bordered="false">
                        {{ linkStateBadge(log) }}
                      </n-tag>
                    </n-flex>
                    <n-flex v-if="log.uploadTime">
                      <n-text
                        >上传于：{{ dayjs.unix(log.uploadTime).format('YYYY-MM-DD HH:mm') }}</n-text
                      >
                    </n-flex>
                    <n-flex>
                      <n-text>创建于：{{ dayjs.unix(log.createdAt).format('YYYY-MM-DD') }}</n-text>
                      <n-tag size="small" :bordered="false">
                        {{ dayjs.unix(log.createdAt).fromNow() }}
                      </n-tag>
                    </n-flex>
                    <n-flex>
                      <n-text>更新于：{{ dayjs.unix(log.updatedAt).format('YYYY-MM-DD') }}</n-text>
                      <n-tag size="small" :bordered="false">
                        {{ dayjs.unix(log.updatedAt).fromNow() }}
                      </n-tag>
                    </n-flex>
                  </n-flex>
                </FoldableCard>
              </template>
            </ListPanel>

            <div
              v-if="
                shouldShowListPagination({
                  total: queryLogPage.total,
                  page: queryLogPage.pageNum,
                  pageSize: queryLogPage.pageSize,
                })
              "
              class="story-pagination-block"
            >
              <n-pagination
                v-model:page="queryLogPage.pageNum"
                v-model:page-size="queryLogPage.pageSize"
                show-size-picker
                :page-sizes="[10, 20, 30, 50]"
                :page-slot="isMobile ? 3 : 5"
                :item-count="queryLogPage.total"
                @update:page="handleLogPageChange"
                @update:page-size="handlePageSizeChange"
              />
            </div>
          </ListWorkspace>
        </template>

        <template v-else-if="mode === 'painter' && currentPainterLog">
          <StoryPainterViewer :log="currentPainterLog" @back="closeItem" />
        </template>

        <template v-else>
          <n-card title="跑团日志 / Story">
            <template #header-extra>
              <n-button type="primary" @click="closeItem">
                <template #icon>
                  <n-icon><i-tabler-arrow-left /></n-icon>
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
                        :swatches="[
                          '#dc2626',
                          '#ea580c',
                          '#ca8a04',
                          '#16a34a',
                          '#0891b2',
                          '#2563eb',
                          '#9333ea',
                          '#db2777',
                        ]"
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
                  <span>{{ line }}</span
                  ><br />
                </template>
              </p>
            </template>
          </div>

          <div
            v-if="
              shouldShowListPagination({
                total: logItemPage.size,
                page: logItemPage.pageNum,
                pageSize: logItemPage.pageSize,
              })
            "
            class="story-pagination"
          >
            <n-pagination
              v-model:page="logItemPage.pageNum"
              v-model:page-size="logItemPage.pageSize"
              show-size-picker
              :page-sizes="[50, 100, 200]"
              :page-slot="isMobile ? 3 : 5"
              :item-count="logItemPage.size"
              @update:page="handleItemPageChange"
              @update:page-size="handleItemPageSizeChange"
            />
          </div>
        </template>
      </n-tab-pane>

      <n-tab-pane tab="日志清理" name="cleanup">
        <section class="story-cleanup-page">
          <TipBox type="info">
            按「超过 N 个月未更新」筛选日志并批量删除。清理只影响日志库，不影响 v1 接口。
          </TipBox>

          <SettingCategoryBox title="清理参数" padded :columns="2">
            <template #title-extra>
              <n-button size="small" secondary @click="refreshCleanupPreview">刷新预览</n-button>
            </template>

            <n-form-item label="未更新月数">
              <n-input-number
                v-model:value="cleanupForm.months"
                :min="0"
                class="cleanup-months"
                :input-props="{ 'aria-label': '未更新月数' }"
              />
            </n-form-item>
            <n-form-item label="执行 VACUUM">
              <n-switch v-model:value="cleanupForm.vacuum" aria-label="执行 VACUUM" />
            </n-form-item>
          </SettingCategoryBox>

          <SettingCategoryBox title="预览结果" padded>
            <div class="cleanup-stats">
              <n-statistic label="待删日志" :value="cleanupPreview?.logs ?? 0" />
              <n-statistic label="待删消息" :value="cleanupPreview?.items ?? 0" />
              <n-statistic
                label="最早更新时间"
                :value="
                  cleanupPreview?.oldestUpdated
                    ? dayjs.unix(cleanupPreview.oldestUpdated).format('YYYY-MM-DD')
                    : '--'
                "
              />
              <n-statistic
                label="最近更新时间"
                :value="
                  cleanupPreview?.newestUpdated
                    ? dayjs.unix(cleanupPreview.newestUpdated).format('YYYY-MM-DD')
                    : '--'
                "
              />
            </div>
          </SettingCategoryBox>

          <SettingCategoryBox title="执行清理" padded>
            <template #notes>
              <TipBox type="warning">
                <p>清理不可撤销，执行前请确认预览结果。</p>
                <p v-if="cleanupForm.vacuum">
                  已开启 VACUUM，这可能导致海豹记录 log 的用户运行缓慢。
                </p>
              </TipBox>
            </template>

            <div class="cleanup-actions">
              <n-button
                :loading="cleanupMutation.isPending.value"
                type="error"
                @click="openCleanupDialog"
              >
                确认并执行
              </n-button>
            </div>
          </SettingCategoryBox>
        </section>
      </n-tab-pane>

      <n-tab-pane tab="日志备份" name="backup">
        <StoryBackup />
      </n-tab-pane>
    </n-tabs>
  </main>
</template>

<script setup lang="tsx">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import { NButton, NFlex, NText, useDialog, useMessage } from 'naive-ui';
import { createProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import QueryToolbar from '@/components/shared/QueryToolbar.vue';
import ResultToolbar from '@/components/shared/ResultToolbar.vue';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import ListPanel from '@/components/shared/ListPanel.vue';
import ListWorkspace from '@/components/shared/ListWorkspace.vue';
import { shouldShowListPagination } from '@/components/shared/listPagination';
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
import PageHeader from '@/components/shared/PageHeader.vue';
import { hasAccessToken } from '@/features/auth/state';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';
import { storyInfoQueryKey } from '@/features/story/queryKeys';
import { summarizeStoryLogs } from '@/features/story/deleteSummary';
import { getStoryPageSizeChange } from '@/features/story/pagination';
import { setStoryLogsSelected } from '@/features/story/selection';
import TipBox from '@/components/shared/TipBox.vue';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');
const StoryPainterViewer = defineAsyncComponent(
  () => import('@/components/storyPainter/StoryPainterViewer.vue')
);

// 跑团日志页包含三类工作流：
// 1. 日志列表与条目查看；
// 2. 日志上传/强制重传，拿到后端链接；
// 3. 旧日志清理和备份管理。
// 数据量可能较大，所以列表和条目分页状态分开维护。
type StoryTab = 'list' | 'cleanup' | 'backup';
type StoryMode = 'logs' | 'items' | 'painter';

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

type StorySearchFormValues = {
  name: string;
  groupId: string;
  createdTime: [number, number] | null;
};

const defaultStorySearchFormValues = (): StorySearchFormValues => ({
  name: '',
  groupId: '',
  createdTime: null,
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
const allLogsSelected = computed(
  () => logs.value.length > 0 && logs.value.every(item => item.pitch)
);
const selectedCount = computed(() => logs.value.filter(item => item.pitch).length);
const itemData = ref<LogOneItem[]>([]);
const currentPainterLog = ref<LogView | null>(null);
const users = ref<Record<string, [string, string]>>({});

const storyInfoQuery = useQuery({
  ...getSdApiV2StoryInfoOptions(),
  enabled: hasAccessToken,
});

const summary = computed(() => storyInfoQuery.data.value?.item);

const storySearchForm = createProSearchForm<StorySearchFormValues>({
  initialValues: cloneSearchFormValues(defaultStorySearchFormValues()),
  onSubmit: async values => {
    Object.assign(queryLogPage.value, {
      ...values,
      pageNum: 1,
    });
    await searchLogs();
  },
  onReset: async () => {
    Object.assign(queryLogPage.value, {
      ...defaultStorySearchFormValues(),
      pageNum: 1,
    });
    await searchLogs();
  },
});

const storySearchColumns: ProSearchFormColumns<StorySearchFormValues> = [
  {
    label: '日志名',
    path: 'name',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索日志名',
    },
  },
  {
    label: '群号',
    path: 'groupId',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索群号',
    },
  },
  {
    label: '日期范围',
    path: 'createdTime',
    field: 'date-range',
    fieldProps: {
      clearable: true,
    },
  },
];

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

function linkStateBadge(log: LogView): string {
  switch (log.linkState) {
    case 'fresh':
      return '已最新';
    case 'stale':
      return '已过期';
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
  // 查询参数在这里从页面状态转换为后端需要的 Unix 秒时间范围。
  // 页面保持毫秒时间戳，方便直接喂给 Naive UI DatePicker。
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
    queryClient.invalidateQueries({ queryKey: storyInfoQueryKey() }),
    searchLogs(),
  ]);
};

function toggleSelectAll() {
  setStoryLogsSelected(logs.value, !allLogsSelected.value);
}

const deleteLogMutation = useMutation({
  mutationFn: async (log: LogView) => {
    const { data } = await deleteSdApiV2StoryLog({
      body: {
        id: log.id,
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
        id: log.id,
        force,
      },
      throwOnError: true,
    });
    return data.item;
  },
});

const cleanupMutation = useMutation({
  mutationFn: async (payload: { months: number; vacuum: boolean }) => {
    const { data } = await postSdApiV2StoryCleanup({
      body: payload,
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
    content: () => (
      <NFlex vertical>
        <NText>此操作不可撤销，将永久删除以下日志：</NText>
        <NText strong>{summarizeStoryLogs([log])}</NText>
      </NFlex>
    ),
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
  const selected = logs.value.filter(item => item.pitch);
  if (selected.length === 0) return;
  dialog.warning({
    title: '删除',
    content: () => (
      <NFlex vertical>
        <NText>此操作不可撤销，将永久删除以下日志：</NText>
        <NText strong>{summarizeStoryLogs(selected)}</NText>
      </NFlex>
    ),
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
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

function batchUploadLogs() {
  const selected = logs.value.filter(item => item.pitch);
  if (selected.length === 0) return;
  dialog.warning({
    title: '批量提取日志',
    content: `将上传所选 ${selected.length} 份日志到海豹服务器，是否继续？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      let ok = 0;
      for (const log of selected) {
        try {
          const result = await uploadLogMutation.mutateAsync({ log, force: false });
          showUploadResult(log, result);
          ok += 1;
        } catch {
          // 单条失败由全局错误提示承载，继续处理其余项。
        }
      }
      if (ok > 0) message.success(`已处理 ${ok} 份日志`);
    },
  });
}

function showUploadResult(
  log: LogView,
  result: { url: string; reused: boolean; forced: boolean; unofficial: boolean }
) {
  message.success(() => (
    <NFlex vertical>
      <NText>
        {result.reused ? '复用已有日志链接' : result.forced ? '强制重新上传成功' : '日志上传成功'}
      </NText>
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
    content: force
      ? '将重新上传此跑团日志并覆盖当前链接记录，是否继续？'
      : '将此跑团日志上传至海豹服务器？',
    positiveText: '确定',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      const result = await uploadLogMutation.mutateAsync({ log, force });
      showUploadResult(log, result);
    },
  });
}

function logActionOptions(log: LogView) {
  return [
    { label: '查看文本明细', key: 'raw' },
    { label: '复制链接', key: 'copy-link', disabled: !log.uploadUrl },
    { label: '强制重传', key: 'force-upload' },
    { type: 'divider', key: 'divider' },
    { label: '删除', key: 'delete' },
  ];
}

function handleLogAction(key: string, log: LogView) {
  if (key === 'raw') void openRawItem(log);
  if (key === 'copy-link' && log.uploadUrl) void copyLink(log.uploadUrl);
  if (key === 'force-upload') uploadLog(log, true);
  if (key === 'link' && log.uploadUrl) openLink(log.uploadUrl);
  if (key === 'delete') delLog(log);
}

async function copyLink(url: string) {
  try {
    await navigator.clipboard.writeText(url);
    message.success('已复制链接到剪贴板');
  } catch {
    message.error('复制失败，请手动复制');
  }
}

async function openItem(log: LogView) {
  const { getStoryPainterAdvancedModeSupport } = await import('@/features/storyPainter/compat');
  const support = getStoryPainterAdvancedModeSupport();
  if (!support.supported) {
    message.warning(support.reason ?? '当前浏览器不支持高级日志模式，已切换到分页文本');
    await openRawItem(log);
    return;
  }
  currentPainterLog.value = log;
  mode.value = 'painter';
}

async function openRawItem(log: LogView) {
  logItemPage.value.logId = log.id;
  logItemPage.value.size = log.size ?? 0;
  logItemPage.value.pageNum = 1;
  await loadStoryItems();
  mode.value = 'items';
}

async function loadStoryItems() {
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

async function handleItemPageChange(value: number) {
  logItemPage.value.pageNum = value;
  await loadStoryItems();
}

async function handleItemPageSizeChange(value: number) {
  Object.assign(logItemPage.value, getStoryPageSizeChange(value));
  await loadStoryItems();
}

function closeItem() {
  itemData.value = [];
  mode.value = 'logs';
  currentPainterLog.value = null;
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
        <NText>
          将删除 {preview.logs} 份超过 {months} 个月未更新的日志，共 {preview.items} 条消息。
        </NText>
        {cleanupForm.value.vacuum ? (
          <NText type="warning">这将可能导致海豹记录 log 用户运行缓慢，请注意</NText>
        ) : null}
      </NFlex>
    ),
    positiveText: '执行清理',
    negativeText: '取消',
    onPositiveClick: executeCleanup,
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
  message.success(
    `已删除 ${result.logs} 份日志、${result.items} 条消息${result.vacuumed ? '，并执行 VACUUM' : ''}`
  );
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

<style scoped>
.story-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.story-summary {
  column-gap: var(--sd-space-lg);
}

.story-selected-count {
  font-size: 0.85rem;
}

.story-log-card {
  width: 100%;
}

.story-item-list {
  margin: 1rem 0;
  padding: 0 1rem;
  overflow-wrap: anywhere;
}

.story-pagination-block {
  display: flex;
  justify-content: flex-end;
}

.story-pagination {
  display: flex;
  justify-content: flex-end;
}

.story-cleanup-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cleanup-months {
  width: min(100%, 12rem);
}

.cleanup-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
  gap: var(--sd-space-md);
}

.cleanup-actions {
  display: flex;
}

@media screen and (max-width: 639.9px) {
  .cleanup-months {
    width: 100%;
  }

  .cleanup-actions :deep(.n-button) {
    width: 100%;
  }
}

@media screen and (max-width: 700px) {
  .story-pagination,
  .story-pagination-block {
    justify-content: flex-start;
  }
}
</style>
