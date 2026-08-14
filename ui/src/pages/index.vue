<template>
  <main class="home-page sd-page-flow">
    <PageHeader title="运行概览">
      <n-tooltip v-if="hasNewVersion && isContainerMode">
        <template #trigger>
          <n-button type="primary" disabled>升级新版</n-button>
        </template>
        容器模式下禁止直接更新，请手动拉取最新镜像
      </n-tooltip>
      <n-button v-else-if="hasNewVersion" type="primary" disabled>升级新版</n-button>
    </PageHeader>

    <section class="overview-grid" aria-label="实例状态">
      <article class="status-card">
        <div class="status-card__heading">
          <span class="status-card__icon">
            <n-icon><i-tabler-cpu /></n-icon>
          </span>
          <span>内存占用</span>
        </div>
        <div class="status-card__value">{{ filesize(memoryUsed) }}</div>
        <div class="status-card__footer">
          <span>运行环境</span>
          <strong>{{ runtimeSummary }}</strong>
        </div>
      </article>

      <article class="status-card group-card">
        <div class="status-card__heading">
          <span class="status-card__icon">
            <n-icon><i-tabler-users-group /></n-icon>
          </span>
          <span>群组服务</span>
        </div>

        <div class="group-metrics">
          <div class="group-metric">
            <strong>{{ groupSummary?.joined ?? '—' }}</strong>
            <span>已加入</span>
          </div>
          <div class="group-metric group-metric--enabled">
            <strong>{{ groupSummary?.enabled ?? '—' }}</strong>
            <span>已启用</span>
          </div>
        </div>
      </article>

      <article class="status-card network-card">
        <div class="status-card__heading">
          <span class="status-card__icon">
            <n-icon><i-tabler-world /></n-icon>
          </span>
          <span>网络质量</span>
          <div class="network-card__actions">
            <n-tooltip v-if="networkHealth.timestamp !== 0">
              <template #trigger>
                <span class="checked-time">
                  {{ formatNetworkHealthRelativeTime(networkHealth.timestamp) }}检测
                </span>
              </template>
              {{ formatNetworkHealthTimestamp(networkHealth.timestamp) }}
            </n-tooltip>
            <n-tooltip>
              <template #trigger>
                <n-button
                  quaternary
                  circle
                  size="tiny"
                  aria-label="重新检测网络质量"
                  :loading="networkHealthRefreshing"
                  :disabled="networkHealthRefreshing"
                  @click="refreshNetworkHealth"
                >
                  <template #icon>
                    <n-icon><i-tabler-refresh /></n-icon>
                  </template>
                </n-button>
              </template>
              重新检测
            </n-tooltip>
          </div>
        </div>

        <div class="network-result">
          <template v-if="networkHealth.timestamp === 0">
            <strong class="network-result__value is-primary">检测中</strong>
            <span>正在检查外部服务连通性</span>
          </template>
          <template
            v-else-if="networkHealth.total !== 0 && networkHealth.total === networkHealth.ok.length"
          >
            <strong class="network-result__value is-success">良好</strong>
            <span>所有服务均可正常访问</span>
          </template>
          <template
            v-else-if="networkHealth.ok.includes('sign') && networkHealth.ok.includes('seal')"
          >
            <strong class="network-result__value is-primary">一般</strong>
            <span>核心服务当前可用</span>
          </template>
          <template v-else-if="networkHealth.total !== 0 && networkHealth.ok.length === 0">
            <strong class="network-result__value is-error">中断</strong>
            <span>外部服务暂时不可访问</span>
          </template>
          <template v-else>
            <strong class="network-result__value is-warning">较差</strong>
            <span>部分服务连接异常</span>
          </template>
        </div>

        <div v-if="networkHealth.timestamp !== 0" class="network-health-targets">
          <span v-for="target in networkTargets" :key="target.key" class="network-target">
            <n-icon
              :color="getWebsiteHealthOK(target.key) ? 'var(--sd-success)' : 'var(--sd-error)'"
            >
              <i-tabler-circle-check-filled v-if="getWebsiteHealthOK(target.key)" />
              <i-tabler-circle-x-filled v-else />
            </n-icon>
            {{ target.label }}
          </span>
        </div>
      </article>
    </section>

    <section class="log-panel">
      <div class="log-panel__header">
        <div class="log-title">
          <h2>运行日志</h2>
          <n-tag :type="logStream.connected.value ? 'success' : 'warning'" size="small" round>
            {{ logStream.connected.value ? '实时连接中' : '未连接' }}
          </n-tag>
        </div>
        <div class="log-controls">
          <n-checkbox v-model:checked="displayReverse">最新在前</n-checkbox>
          <n-checkbox v-model:checked="autoRefresh">保持刷新</n-checkbox>
          <n-button size="small" secondary @click="logStream.reconnect">重连</n-button>
          <n-button size="small" type="primary" secondary @click="scrollToLatestLog">
            <template #icon>
              <n-icon>
                <i-tabler-chevron-up v-if="displayReverse" />
                <i-tabler-chevron-down v-else />
              </n-icon>
            </template>
            最新日志
          </n-button>
        </div>
      </div>

      <TipBox v-if="logStream.errorText.value" type="error" class="log-alert">
        {{ logStream.errorText.value }}
      </TipBox>

      <div ref="logsContainer" class="logs">
        <n-data-table
          ref="logTable"
          :data="logData"
          :columns="columns"
          :row-key="getLogRowKey"
          :class="isMobile ? 'w-full' : ''"
          :bordered="false"
          size="small"
          :max-height="isMobile ? 420 : 620"
          :virtual-scroll="!isMobile"
        />
        <n-empty v-if="!logData.length" description="暂无日志" class="empty-log" />
        <n-back-top :right="30" />
      </div>
    </section>
  </main>
</template>

<script setup lang="tsx">
import { computed, ref, useTemplateRef, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useQuery } from '@tanstack/vue-query';
import { filesize } from 'filesize';
import dayjs from 'dayjs';
import type { DataTableColumns, DataTableInst } from 'naive-ui';
import {
  getSdApiV2BaseNetworkHealthOptions,
  getSdApiV2BaseOverviewOptions,
  postSdApiV2GroupList,
} from '@/api';
import { useBaseLogStream, type BaseLogEntry } from '@/features/base/logStream';
import { applyLogDisplayUpdate } from '@/features/base/logDisplayState';
import {
  formatNetworkHealthRelativeTime,
  formatNetworkHealthTimestamp,
  isNetworkHealthTargetOK,
  normalizeNetworkHealthData,
} from '@/features/base/networkHealth';
import { formatRuntimeSummary } from '@/features/base/runtimeSummary';
import { hasAccessToken } from '@/features/auth/state';
import PageHeader from '@/components/shared/PageHeader.vue';
import TipBox from '@/components/shared/TipBox.vue';

const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

// 首页是运行概览页：低频状态走 overview query，高频日志走 realtime。
// 不把日志混进 overview，是为了避免 5 秒轮询带回大量日志数据。
const overviewQuery = useQuery({
  ...getSdApiV2BaseOverviewOptions(),
  enabled: hasAccessToken,
  refetchInterval: 5000,
});

const networkHealthQuery = useQuery({
  ...getSdApiV2BaseNetworkHealthOptions(),
  enabled: hasAccessToken,
  refetchInterval: 300000,
});

const groupSummaryQuery = useQuery({
  queryKey: ['home', 'group-summary'],
  queryFn: fetchGroupSummary,
  enabled: hasAccessToken,
  staleTime: 30000,
  refetchInterval: 60000,
});

const overview = computed(() => overviewQuery.data.value?.item);
const memoryUsed = computed(() => overview.value?.memory.usedSys ?? 0);
const groupSummary = computed(() => groupSummaryQuery.data.value);
const runtimeSummary = computed(() =>
  formatRuntimeSummary(overview.value?.runtime, { withMode: true })
);
const hasNewVersion = computed(() => {
  const version = overview.value?.version;
  if (!version) return false;
  return version.code < version.latestCode;
});
const isContainerMode = computed(() => overview.value?.runtime.containerMode === true);

const displayReverse = ref(true);
const autoRefresh = ref(true);
const visibleLogs = ref<BaseLogEntry[]>([]);
const networkTargets = [
  { key: 'seal', label: '官网' },
  { key: 'sign', label: 'Lagrange Sign' },
  { key: 'google', label: 'Google' },
  { key: 'github', label: 'GitHub' },
];
const logsContainer = useTemplateRef<HTMLElement>('logsContainer');
const logTable = useTemplateRef<DataTableInst>('logTable');
const logStream = useBaseLogStream();
const networkHealth = computed(() =>
  normalizeNetworkHealthData(networkHealthQuery.data.value?.item)
);
const networkHealthRefreshing = computed(() => networkHealthQuery.isFetching.value);

watch(
  [logStream.logs, autoRefresh],
  () => {
    visibleLogs.value = applyLogDisplayUpdate(
      visibleLogs.value,
      logStream.logs.value,
      autoRefresh.value
    );
  },
  { immediate: true }
);

// 日志一些情况下因为ws连接建立早于页面初始化
// 导致日志数据不能传给logStream
// 建议给logStream做一个状态管理让他能保持
// 现在先通过加载页面默认重连解决
logStream.reconnect();

// 日志源保持 append 顺序，展示顺序只在 computed 中转换，避免切换“倒序显示”
// 时破坏原始缓冲和后续 append 逻辑。
const logData = computed(() => {
  return displayReverse.value ? [...visibleLogs.value].reverse() : visibleLogs.value;
});

function getLogRowKey(row: BaseLogEntry) {
  return row.id;
}

async function fetchGroupSummary() {
  const firstPageSize = 100;
  const firstResponse = await postSdApiV2GroupList({
    body: {
      page: 1,
      pageSize: firstPageSize,
      keyword: '',
      filter: {},
    },
    throwOnError: true,
  });

  let total = Number(firstResponse.data.item.total ?? 0);
  let groups = firstResponse.data.item.list ?? [];

  if (groups.length < total) {
    const fullResponse = await postSdApiV2GroupList({
      body: {
        page: 1,
        pageSize: total,
        keyword: '',
        filter: {},
      },
      throwOnError: true,
    });
    total = Number(fullResponse.data.item.total ?? total);
    groups = fullResponse.data.item.list ?? [];
  }

  return {
    joined: total,
    enabled: groups.filter(group => group.active).length,
  };
}

function refreshNetworkHealth() {
  void networkHealthQuery.refetch();
}

function getWebsiteHealthOK(target: string): boolean {
  return isNetworkHealthTargetOK(networkHealth.value, target);
}

// 「最新日志」始终跳到最新那一条：倒序显示时它在顶部，正序显示时在尾部。
// 走表格实例的 scrollTo，因为虚拟滚动下真正的滚动元素是内部的 .v-vl，
// .n-data-table-base-table-body 只是外层 scrollbar 包装，对它 scrollTo 不会生效。
function scrollToLatestLog() {
  if (!logData.value.length) return;

  // 到底部用一个足够大的 top 交给浏览器 clamp：scrollTo 的公开类型里没有
  // position 字段，而它内部实现就是 top: MAX_SAFE_INTEGER。
  const top = displayReverse.value ? 0 : Number.MAX_SAFE_INTEGER;
  const table = logTable.value;
  if (table) {
    table.scrollTo({ top, behavior: 'smooth' });
    return;
  }

  getLogScrollElement()?.scrollTo({ top, behavior: 'smooth' });
}

function getLogScrollElement(): HTMLElement | null {
  const root = logsContainer.value;
  if (!root) return null;
  return (
    root.querySelector<HTMLElement>('.n-data-table-base-table-body .v-vl') ??
    root.querySelector<HTMLElement>('.n-data-table-base-table-body')
  );
}

type LogLevelTone = 'neutral' | 'info' | 'warning' | 'error';

function getLogLevelMeta(level: string): { label: string; tone: LogLevelTone } {
  switch (level.toLowerCase()) {
    case 'info':
      return { label: 'INFO', tone: 'info' };
    case 'warn':
    case 'warning':
      return { label: 'WARN', tone: 'warning' };
    case 'error':
    case 'fatal':
      return { label: level.toUpperCase(), tone: 'error' };
    default:
      return { label: level.toUpperCase() || 'LOG', tone: 'neutral' };
  }
}

function renderLogLevel(row: BaseLogEntry) {
  const meta = getLogLevelMeta(row.level);
  return <span class={['log-level', `log-level--${meta.tone}`]}>{meta.label}</span>;
}

const columns = computed<DataTableColumns<BaseLogEntry>>(() => {
  const data: DataTableColumns<BaseLogEntry> = [
    {
      title: '时间',
      key: 'ts',
      width: isMobile.value ? 70 : 100,
      render: row => (
        <div class="log-time">
          <span class="log-time-text">
            {dayjs.unix(row.ts).format(isMobile.value ? 'HH:mm' : 'HH:mm:ss')}
          </span>
        </div>
      ),
    },
  ];

  data.push({
    title: '级别',
    key: 'level',
    width: isMobile.value ? 68 : 76,
    render: renderLogLevel,
  });

  data.push({
    title: '信息',
    key: 'msg',
    render: row => <div class="log-message">{row.msg}</div>,
  });

  return data;
});
</script>

<style scoped>
.home-page {
  max-width: 1240px;
  margin: 0 auto;
}

h2,
p {
  margin: 0;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.status-card {
  min-width: 0;
  min-height: 176px;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  color: var(--sd-text-primary);
  padding: 1.25rem;
  text-align: left;
  transition:
    border-color var(--sd-transition-base),
    background-color var(--sd-transition-base);
}

.status-card__heading {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  color: var(--sd-text-secondary);
  font-size: 0.82rem;
  font-weight: 600;
}

.status-card__icon {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-selected);
  color: var(--sd-primary);
  font-size: 1rem;
}

.status-card__value {
  margin-top: 1.25rem;
  font-size: 1.9rem;
  font-weight: 650;
  letter-spacing: -0.03em;
  line-height: 1.2;
}

.status-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1.15rem;
  padding-top: 0.8rem;
  border-top: 1px solid var(--sd-border-soft);
  color: var(--sd-text-muted);
  font-size: 0.76rem;
}

.status-card__footer strong {
  overflow: hidden;
  color: var(--sd-text-secondary);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 1.15rem;
}

.group-metric {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.35rem;
}

.group-metric + .group-metric {
  margin-left: 1rem;
  padding-left: 1rem;
  border-left: 1px solid var(--sd-border-soft);
}

.group-metric strong {
  overflow: hidden;
  color: var(--sd-text-primary);
  font-size: 1.9rem;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  letter-spacing: -0.03em;
  line-height: 1.2;
  text-overflow: ellipsis;
}

.group-metric span {
  color: var(--sd-text-muted);
  font-size: 0.76rem;
}

.group-metric--enabled strong {
  color: var(--sd-primary);
}

.network-card {
  width: 100%;
}

.network-card__actions {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-left: auto;
}

.checked-time {
  color: var(--sd-text-muted);
  font-size: 0.72rem;
  font-weight: 400;
}

.network-result {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1.25rem;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
}

.network-result__value {
  color: var(--sd-text-primary);
  font-size: 1.45rem;
  font-weight: 650;
  letter-spacing: -0.02em;
}

.is-primary {
  color: var(--sd-primary);
}
.is-success {
  color: var(--sd-success);
}
.is-warning {
  color: var(--sd-warning);
}
.is-error {
  color: var(--sd-error);
}

.network-health-targets {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.55rem;
  margin-top: 1rem;
}

.network-target {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-xs);
  background: var(--sd-bg-elevated-soft);
  color: var(--sd-text-secondary);
  font-size: 0.72rem;
  padding: 0.22rem 0.45rem;
}

.log-panel {
  overflow: hidden;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

.log-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--sd-border-soft);
}

.log-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.log-title h2 {
  color: var(--sd-text-primary);
  font-size: 1rem;
  font-weight: 650;
}

.log-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: var(--sd-space-sm);
}

.log-alert {
  margin: 1rem 1.25rem 0;
}

.logs {
  min-height: 300px;
}

.empty-log {
  padding: 3rem 0;
}

:deep(.logs .n-data-table-th) {
  background: var(--sd-bg-elevated-soft);
  color: var(--sd-text-secondary);
  font-size: 0.75rem;
  font-weight: 600;
}

:deep(.log-time) {
  display: flex;
  align-items: center;
  color: var(--sd-text-muted);
}

:deep(.log-time-text) {
  font-variant-numeric: tabular-nums;
}

:deep(.log-level) {
  display: inline-flex;
  min-width: 42px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-xs);
  background: var(--sd-bg-elevated-muted);
  color: var(--sd-text-secondary);
  font-family: var(--sd-font-code);
  font-size: 0.67rem;
  font-weight: 500;
  letter-spacing: 0.025em;
  line-height: 1.45;
  padding: 0.08rem 0.32rem;
  white-space: nowrap;
  word-break: keep-all;
}

:deep(.log-level--info) {
  border-color: var(--sd-primary-border);
  background: var(--sd-primary-soft);
  color: var(--sd-primary);
}

:deep(.log-level--warning) {
  border-color: var(--sd-warning-border);
  background: var(--sd-warning-soft);
  color: var(--sd-warning);
  font-weight: 600;
}

:deep(.log-level--error) {
  border-color: var(--sd-error-border);
  background: var(--sd-error-soft);
  color: var(--sd-error);
  font-weight: 600;
}

:deep(.log-message) {
  display: flex;
  align-items: flex-start;
  gap: 0.55rem;
  color: var(--sd-text-primary);
  font-family: var(--sd-font-code);
}

@media (max-width: 1100px) {
  .overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .log-panel__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .overview-grid {
    grid-template-columns: 1fr;
  }

  .log-controls {
    justify-content: flex-start;
  }
}
</style>
