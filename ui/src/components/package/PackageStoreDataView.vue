<template>
  <ResponsiveDataView :compact-at="760" aria-label="扩展包商店">
    <template #table>
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :row-key="rowKey"
        :scroll-x="1040"
        striped
        size="small"
      />
    </template>

    <template #compact>
      <n-spin :show="loading">
        <n-empty v-if="!loading && rows.length === 0" description="暂无商店扩展包" />
        <ul v-else class="store-compact-list">
          <li v-for="row in rows" :key="rowKey(row)" class="store-compact-item">
            <div class="store-compact-item__heading">
              <div class="store-compact-item__identity">
                <strong>{{ row.name }}</strong>
                <span>{{ row.id }} · {{ row.version }}</span>
              </div>
              <n-tag size="small" :type="row.installed ? 'success' : 'default'" :bordered="false">
                {{ row.installed ? '已安装' : '未安装' }}
              </n-tag>
            </div>
            <dl class="store-compact-item__details">
              <div>
                <dt>作者</dt>
                <dd>{{ row.authors?.join(' / ') || '-' }}</dd>
              </div>
              <div>
                <dt>分类</dt>
                <dd>{{ row.storeAssets?.category || '-' }}</dd>
              </div>
              <div>
                <dt>更新</dt>
                <dd>{{ formatUpdateTime(row) }}</dd>
              </div>
            </dl>
            <n-button size="small" type="primary" secondary @click="emit('previewInstall', row)">
              {{ row.installed ? '预览重装' : '预览安装' }}
            </n-button>
          </li>
        </ul>
      </n-spin>
    </template>
  </ResponsiveDataView>
</template>

<script setup lang="tsx">
import { NButton, NTag, type DataTableColumns } from 'naive-ui';
import type { StorePackage } from '@/api';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';

defineProps<{
  rows: StorePackage[];
  loading: boolean;
}>();

const emit = defineEmits<{
  previewInstall: [row: StorePackage];
}>();

const rowKey = (row: StorePackage) => `${row.id}@${row.version}`;
const formatUpdateTime = (row: StorePackage) =>
  new Date((row.download?.updateTime ?? 0) * 1000).toLocaleString();

const columns: DataTableColumns<StorePackage> = [
  {
    title: '扩展包',
    key: 'name',
    minWidth: 260,
    render: row => (
      <div class="store-name-cell">
        <strong>{row.name}</strong>
        <span>
          {row.id} · {row.version}
        </span>
      </div>
    ),
  },
  { title: '作者', key: 'authors', minWidth: 160, render: row => row.authors?.join(' / ') || '-' },
  { title: '分类', key: 'category', width: 120, render: row => row.storeAssets?.category || '-' },
  { title: '更新时间', key: 'updateTime', width: 180, render: formatUpdateTime },
  {
    title: '安装状态',
    key: 'installed',
    width: 110,
    render: row => (
      <NTag size="small" type={row.installed ? 'success' : 'default'} bordered={false}>
        {row.installed ? '已安装' : '未安装'}
      </NTag>
    ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    render: row => (
      <NButton size="small" text type="primary" onClick={() => emit('previewInstall', row)}>
        {row.installed ? '预览重装' : '预览安装'}
      </NButton>
    ),
  },
];
</script>

<style scoped>
:deep(.store-name-cell),
.store-compact-item__identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.2rem;
}

:deep(.store-name-cell span),
.store-compact-item__identity span {
  color: var(--sd-text-secondary);
  font-size: 0.78rem;
  overflow-wrap: anywhere;
}

.store-compact-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.75rem;
  list-style: none;
}

.store-compact-item {
  display: grid;
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: 6px;
  background: var(--sd-bg-elevated);
  gap: 0.75rem;
  padding: 0.875rem;
}

.store-compact-item__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.store-compact-item__details {
  display: grid;
  margin: 0;
  gap: 0.45rem;
}

.store-compact-item__details div {
  display: grid;
  grid-template-columns: 3rem minmax(0, 1fr);
  gap: 0.5rem;
}

.store-compact-item__details dt {
  color: var(--sd-text-muted);
}

.store-compact-item__details dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
