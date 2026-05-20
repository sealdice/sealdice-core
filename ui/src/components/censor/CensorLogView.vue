<script setup lang="tsx">
import type { DataTableColumns } from 'naive-ui';
import type { CensorLog } from '@/api';
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
    render: row => <CensorSensitiveTag level={row.highestLevel} />,
  },
  {
    title: '消息类型',
    key: 'msgType',
    render: row => <n-text>{formatCensorMessageType(row.msgType)}</n-text>,
  },
  { title: '用户', key: 'userId' },
  { title: '群', key: 'groupId' },
  { title: '内容', key: 'content' },
  {
    title: '消息时间',
    key: 'createdAt',
    render: row => <>{formatCensorLogTime(row.createdAt)}</>,
  },
];
</script>

<template>
  <div class="censor-log-container">
    <header class="censor-log-header">
      <n-button type="info" secondary @click="emit('refresh')">
        <template #icon>
          <n-icon><i-carbon-renew /></n-icon>
        </template>
        刷新
      </n-button>
      <n-pagination
        size="small"
        v-model:page="query.pageNum"
        v-model:page-size="query.pageSize"
        :item-count="total"
        :page-slot="5"
        :default-page-size="20"
      />
    </header>
    <n-spin :show="loading">
      <n-data-table :columns="columns" :data="logs" class="mt-4" />
    </n-spin>
  </div>
</template>

<style scoped>
.censor-log-container {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.censor-log-header {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  text-align: center;
  flex-wrap: wrap;
  gap: 1rem;
}
</style>
