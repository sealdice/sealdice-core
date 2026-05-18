<script setup lang="tsx">
import { computed, ref } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useQuery } from '@tanstack/vue-query';
import { filesize } from 'filesize';
import dayjs from 'dayjs';
import type { DataTableColumns } from 'naive-ui';
import { useThemeVars } from 'naive-ui';
import {
  getSdApiV2BaseOverviewOptions,
} from '@/api';
import { useBaseLogStream, type BaseLogItem } from '@/features/base/logStream';
import { hasAccessToken } from '@/features/auth/state';

const themeVars = useThemeVars();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

// 首页是运行概览页：低频状态走 overview query，高频日志走 realtime。
// 不把日志混进 overview，是为了避免 5 秒轮询带回大量日志数据。
const overviewQuery = useQuery({
  ...getSdApiV2BaseOverviewOptions(),
  enabled: hasAccessToken,
  refetchInterval: 5000,
});

const overview = computed(() => overviewQuery.data.value?.item);
const memoryUsed = computed(() => overview.value?.memory.usedSys ?? 0);
const hasNewVersion = computed(() => {
  const version = overview.value?.version;
  if (!version) return false;
  return version.code < version.latestCode;
});
const isContainerMode = computed(() => overview.value?.runtime.containerMode === true);

const displayReverse = ref(true);
const autoRefresh = ref(true);
const logStream = useBaseLogStream();
// 日志源保持 append 顺序，展示顺序只在 computed 中转换，避免切换“倒序显示”
// 时破坏原始缓冲和后续 append 逻辑。
const logData = computed(() => {
  return displayReverse.value ? [...logStream.logs.value].reverse() : logStream.logs.value;
});

const getMsgColor = (row: BaseLogItem): string | undefined => {
  if (row.msg.startsWith('onebot | ')) return themeVars.value.warningColor;
  if (row.msg.startsWith('发给')) return themeVars.value.infoColor;
  if (row.level === 'warn') return themeVars.value.warningColor;
  if (row.level === 'error') return themeVars.value.errorColor;
  return undefined;
};

const columns = computed<DataTableColumns<BaseLogItem>>(() => {
  const data: DataTableColumns<BaseLogItem> = [
    {
      title: '时间',
      key: 'ts',
      width: isMobile.value ? 70 : 100,
      render: row => {
        const color = getMsgColor(row);
        return (
          <div class='log-time' style={{ color }}>
            {isMobile.value ? null : (
              <n-icon>
                <i-carbon-time />
              </n-icon>
            )}
            <span class='log-time-text'>
              {dayjs.unix(row.ts).format(isMobile.value ? 'HH:mm' : 'HH:mm:ss')}
            </span>
          </div>
        );
      },
    },
  ];

  if (!isMobile.value) {
    data.push({
      title: '级别',
      key: 'level',
      width: 70,
      render: row => {
        const color = getMsgColor(row);
        return <span style={{ color }}>{row.level}</span>;
      },
    });
  }

  data.push({
    title: '信息',
    key: 'msg',
    render: row => {
      const color = getMsgColor(row);
      return <span style={{ color }}>{row.msg}</span>;
    },
  });

  return data;
});
</script>

<template>
  <main class="home-page">
    <div class="upgrade-bar">
      <n-tooltip v-if="hasNewVersion && isContainerMode">
        <template #trigger>
          <n-button type="primary" disabled>
            升级新版
          </n-button>
        </template>
        容器模式下禁止直接更新，请手动拉取最新镜像
      </n-tooltip>
      <n-button v-else-if="hasNewVersion" type="primary" disabled>
        升级新版
      </n-button>
    </div>

    <h4>状态</h4>
    <div class="status-block">
      <div class="status-line">
        <n-text>内存占用：</n-text>
        <n-text class="memory-value">
          {{ filesize(memoryUsed) }}
        </n-text>
        <n-text type="info" class="memory-tip">
          理论内存占用，数值偏大。系统任务管理器中的「活动内存」才是实际使用的系统内存。
        </n-text>
      </div>

      <n-text type="info" class="memory-tip">
        运行环境：{{ overview ? `${overview.runtime.OS} - ${overview.runtime.arch}` : '读取中' }}
      </n-text>
    </div>

    <div class="log-head">
      <h4>日志</h4>
      <div class="log-controls">
        <n-tag :type="logStream.connected.value ? 'success' : 'warning'" size="small">
          {{ logStream.connected.value ? '实时连接中' : '未连接' }}
        </n-tag>
        <n-button size="small" secondary @click="logStream.reconnect">
          重连
        </n-button>
        <n-checkbox v-model:checked="displayReverse">
          最新日志展示在最上方
        </n-checkbox>
        <n-checkbox v-model:checked="autoRefresh">
          保持刷新
        </n-checkbox>
      </div>
    </div>

    <n-alert v-if="logStream.errorText.value" type="error" class="log-alert">
      {{ logStream.errorText.value }}
    </n-alert>

    <main class="logs">
      <n-data-table
        :data="logData"
        :columns="columns"
        :class="isMobile ? 'w-full' : ''"
        :bordered="false"
        size="small"
      />

      <n-empty v-if="!logStream.hasLogs.value" description="暂无日志" class="empty-log" />
      <n-back-top :right="30" />
    </main>
  </main>
</template>

<style scoped>
.home-page {
  max-width: 1180px;
  margin: 0 auto;
}

.upgrade-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

h4 {
  margin: 1rem 0 0.75rem;
  color: #111827;
  font-size: 1rem;
  font-weight: 700;
}

.status-block {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 1rem;
}

.status-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.25rem;
}

.memory-value {
  margin-right: 0.5rem;
}

.memory-tip {
  font-size: 0.75rem;
}

.log-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1rem;
}

.log-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
}

.log-alert {
  margin-bottom: 1rem;
}

.logs {
  padding-bottom: 2rem;
}

.empty-log {
  padding: 2rem 0;
}

:deep(.log-time) {
  display: flex;
  align-items: center;
}

:deep(.log-time-text) {
  margin-left: 0.25rem;
}

@media (max-width: 720px) {
  .log-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .log-controls {
    justify-content: flex-start;
  }
}
</style>
