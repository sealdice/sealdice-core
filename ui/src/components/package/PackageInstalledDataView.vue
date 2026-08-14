<template>
  <ResponsiveDataView :compact-at="760" aria-label="已安装扩展包">
    <template #table>
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :row-key="rowKey"
        :scroll-x="1160"
        :bordered="false"
        striped
        size="small"
      />
    </template>

    <template #compact>
      <n-spin :show="loading">
        <n-empty v-if="!loading && rows.length === 0" description="暂无已安装扩展包" />
        <ul v-else class="package-compact-list">
          <li v-for="row in rows" :key="rowKey(row)" class="package-compact-item">
            <div class="package-compact-item__heading">
              <div class="package-compact-item__identity">
                <strong>{{ row.manifest.package.name }}</strong>
                <span>{{ row.manifest.package.id }} · {{ row.manifest.package.version }}</span>
              </div>
              <n-tag
                size="small"
                :type="row.state === 'enabled' ? 'success' : 'warning'"
                :bordered="false"
              >
                {{ row.state === 'enabled' ? '已启用' : row.state }}
              </n-tag>
            </div>

            <dl class="package-compact-item__details">
              <div>
                <dt>内容</dt>
                <dd>{{ contentLabels(row).join('、') || '未声明' }}</dd>
              </div>
              <div>
                <dt>来源</dt>
                <dd>{{ sourceLabel(row) }}</dd>
              </div>
            </dl>

            <n-flex size="small" wrap>
              <n-button size="small" secondary @click="emit('detail', row)">详情</n-button>
              <n-button
                size="small"
                secondary
                @click="emit('toggle', row, row.state !== 'enabled')"
              >
                {{ row.state === 'enabled' ? '禁用' : '启用' }}
              </n-button>
              <n-button size="small" secondary @click="emit('reload', row)">重载</n-button>
              <n-button size="small" secondary type="error" @click="emit('uninstall', row)">
                卸载
              </n-button>
            </n-flex>
          </li>
        </ul>
      </n-spin>
    </template>
  </ResponsiveDataView>
</template>

<script setup lang="tsx">
import { NButton, NFlex, NTag, type DataTableColumns } from 'naive-ui';
import type { Instance } from '@/api';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';

defineProps<{
  rows: Instance[];
  loading: boolean;
}>();

const emit = defineEmits<{
  detail: [row: Instance];
  toggle: [row: Instance, enabled: boolean];
  reload: [row: Instance];
  uninstall: [row: Instance];
}>();

const rowKey = (row: Instance) => row.manifest.package.id;

function contentLabels(row: Instance): string[] {
  return Object.entries(row.manifest.contents ?? {})
    .filter(([, items]) => Array.isArray(items) && items.length > 0)
    .map(([key]) => key);
}

function sourceLabel(row: Instance): string {
  const status = row.sourceStatus === 'cache_only' ? '仅缓存' : '原始包存在';
  const detail = row.sourceWarning || row.sourcePath;
  return detail ? `${status} · ${detail}` : status;
}

const columns: DataTableColumns<Instance> = [
  {
    title: '扩展包',
    key: 'name',
    minWidth: 260,
    render: row => (
      <div class="package-name-cell">
        <strong>{row.manifest.package.name}</strong>
        <span>
          {row.manifest.package.id} · {row.manifest.package.version}
        </span>
      </div>
    ),
  },
  {
    title: '状态',
    key: 'state',
    width: 110,
    render: row => (
      <NTag size="small" type={row.state === 'enabled' ? 'success' : 'warning'} bordered={false}>
        {row.state === 'enabled' ? '已启用' : row.state}
      </NTag>
    ),
  },
  {
    title: '内容',
    key: 'contents',
    minWidth: 180,
    render: row => (
      <NFlex size="small" wrap>
        {contentLabels(row).map(label => (
          <NTag key={label} size="small" bordered={false}>
            {label}
          </NTag>
        ))}
      </NFlex>
    ),
  },
  {
    title: '来源',
    key: 'source',
    minWidth: 220,
    render: row => sourceLabel(row),
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
    render: row => (
      <NFlex size="small" wrap>
        <NButton size="small" text type="primary" onClick={() => emit('detail', row)}>
          详情
        </NButton>
        <NButton
          size="small"
          text
          type="primary"
          onClick={() => emit('toggle', row, row.state !== 'enabled')}
        >
          {row.state === 'enabled' ? '禁用' : '启用'}
        </NButton>
        <NButton size="small" text type="primary" onClick={() => emit('reload', row)}>
          重载
        </NButton>
        <NButton size="small" text type="error" onClick={() => emit('uninstall', row)}>
          卸载
        </NButton>
      </NFlex>
    ),
  },
];
</script>

<style scoped>
:deep(.package-name-cell),
.package-compact-item__identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.2rem;
}

:deep(.package-name-cell span),
.package-compact-item__identity span {
  color: var(--sd-text-secondary);
  font-size: 0.78rem;
  overflow-wrap: anywhere;
}

.package-compact-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.75rem;
  list-style: none;
}

.package-compact-item {
  display: grid;
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  gap: 0.75rem;
  padding: 0.875rem;
}

.package-compact-item__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.package-compact-item__details {
  display: grid;
  margin: 0;
  gap: 0.45rem;
}

.package-compact-item__details div {
  display: grid;
  grid-template-columns: 3rem minmax(0, 1fr);
  gap: 0.5rem;
}

.package-compact-item__details dt {
  color: var(--sd-text-muted);
}

.package-compact-item__details dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
