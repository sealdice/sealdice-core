<template>
  <ListWorkspace class="censor-log-container">
    <ListPanel>
      <template #toolbar>
        <ResultToolbar>
          <template #meta>
            <n-text depth="3">共 {{ total }} 项，本页 {{ logs.length }} 项</n-text>
          </template>
          <n-button secondary size="small" @click="emit('refresh')">
            <template #icon>
              <n-icon><i-tabler-refresh /></n-icon>
            </template>
            刷新
          </n-button>
        </ResultToolbar>
      </template>
      <n-spin :show="loading">
        <ListEmptyState v-if="!loading && logs.length === 0" description="暂无拦截命中日志" />
        <ResponsiveDataView v-else :compact-at="960" aria-label="拦截命中日志">
          <template #table>
            <n-data-table
              :columns="columns"
              :data="logs"
              :scroll-x="940"
              :bordered="false"
              size="small"
            />
          </template>
          <template #compact>
            <ul class="censor-log-list">
              <li v-for="log in logs" :key="log.id" class="censor-log-list__item">
                <div class="censor-log-list__heading">
                  <n-flex size="small" align="center" wrap>
                    <CensorSensitiveTag :level="log.highestLevel" />
                    <n-tag size="small" :bordered="false">
                      {{ formatCensorMessageType(log.msgType) }}
                    </n-tag>
                  </n-flex>
                  <time>{{ formatCensorLogTime(log.createdAt) }}</time>
                </div>
                <dl>
                  <div>
                    <dt>用户</dt>
                    <dd>{{ log.userId || '-' }}</dd>
                  </div>
                  <div>
                    <dt>群组</dt>
                    <dd>{{ log.groupId || '-' }}</dd>
                  </div>
                </dl>
                <p>{{ log.content }}</p>
              </li>
            </ul>
          </template>
        </ResponsiveDataView>
      </n-spin>
    </ListPanel>
    <footer
      v-if="shouldShowListPagination({ total, page: query.pageNum, pageSize: query.pageSize })"
      class="censor-log-footer"
    >
      <n-pagination
        v-model:page="query.pageNum"
        v-model:page-size="query.pageSize"
        :item-count="total"
        :page-slot="3"
        show-size-picker
        :page-sizes="[10, 20, 30, 50]"
        @update:page-size="query.pageNum = 1"
      />
    </footer>
  </ListWorkspace>
</template>

<script setup lang="tsx">
import type { DataTableColumns } from 'naive-ui';
import type { CensorLog } from '@/api';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import ListPanel from '@/components/shared/ListPanel.vue';
import ListWorkspace from '@/components/shared/ListWorkspace.vue';
import { shouldShowListPagination } from '@/components/shared/listPagination';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';
import ResultToolbar from '@/components/shared/ResultToolbar.vue';
import type { CensorLogQueryModel } from '@/features/censor/viewModel';
import CensorSensitiveTag from './CensorSensitiveTag.vue';
import { formatCensorLogTime, formatCensorMessageType } from '@/features/censor/viewModel';

const query = defineModel<CensorLogQueryModel>('query', { required: true });

defineProps<{
  logs: CensorLog[];
  total: number;
  loading: boolean;
}>();

const emit = defineEmits<{
  refresh: [];
}>();

const columns: DataTableColumns<CensorLog> = [
  {
    title: '命中级别',
    key: 'highestLevel',
    minWidth: 120,
    render: row => <CensorSensitiveTag level={row.highestLevel} />,
  },
  {
    title: '消息类型',
    key: 'msgType',
    minWidth: 110,
    render: row => <n-text>{formatCensorMessageType(row.msgType)}</n-text>,
  },
  { title: '用户', key: 'userId', minWidth: 130, ellipsis: { tooltip: true } },
  { title: '群', key: 'groupId', minWidth: 130, ellipsis: { tooltip: true } },
  { title: '内容', key: 'content', minWidth: 280, ellipsis: { tooltip: true } },
  {
    title: '消息时间',
    key: 'createdAt',
    minWidth: 170,
    render: row => <>{formatCensorLogTime(row.createdAt)}</>,
  },
];
</script>

<style scoped>
.censor-log-footer {
  display: flex;
  justify-content: flex-end;
}

.censor-log-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.75rem;
  list-style: none;
}

.censor-log-list__item {
  display: grid;
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  gap: 0.65rem;
  padding: 0.75rem;
}

.censor-log-list__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.censor-log-list__heading time {
  color: var(--sd-text-muted);
  font-size: 0.78rem;
  white-space: nowrap;
}

.censor-log-list__item dl {
  display: grid;
  margin: 0;
  gap: 0.35rem;
}

.censor-log-list__item dl div {
  display: grid;
  grid-template-columns: 3rem minmax(0, 1fr);
  gap: 0.5rem;
}

.censor-log-list__item dt {
  color: var(--sd-text-muted);
}

.censor-log-list__item dd,
.censor-log-list__item p {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
