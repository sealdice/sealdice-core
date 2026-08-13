<template>
  <section class="backup-file-list">
    <div class="backup-file-list__head backup-file-list__summary">
      <n-text strong>已备份文件</n-text>
      <n-text depth="3">{{ items.length }} 个备份文件</n-text>
    </div>
    <div class="backup-file-list__head backup-file-list__toolbar">
      <n-button type="primary" :disabled="disabled" @click="emit('openBackup')">
        立即备份
      </n-button>
      <n-button
        type="error"
        secondary
        :disabled="disabled || items.length === 0"
        @click="emit('openBatchDelete')"
      >
        <template #icon>
          <n-icon>
            <i-tabler-trash />
          </n-icon>
        </template>
        批量删除
      </n-button>
    </div>

    <n-empty v-if="!loading && items.length === 0" description="暂无备份文件">
      <template #extra>
        <n-button type="primary" :disabled="disabled" @click="emit('openBackup')">
          立即备份
        </n-button>
      </template>
    </n-empty>

    <ResponsiveDataView v-else :compact-at="680" aria-label="备份文件列表">
      <template #table>
        <n-data-table
          :columns="columns"
          :data="items"
          :loading="loading"
          :bordered="false"
          :row-key="(row: FileItem) => row.name"
          :scroll-x="680"
          :max-height="520"
          virtual-scroll
          size="small"
        />
      </template>

      <template #compact>
        <n-spin :show="loading">
          <ul class="backup-file-list__compact-list">
            <li v-for="item in items" :key="item.name" class="backup-file-list__compact-item">
              <div class="backup-file-list__compact-heading">
                <strong>{{ item.name }}</strong>
                <n-tag size="small" :bordered="false">{{ filesize(item.fileSize) }}</n-tag>
              </div>
              <n-text depth="3" class="backup-file-list__compact-selection">
                {{
                  describeBackupSelection(item.selection).length
                    ? `包含：${describeBackupSelection(item.selection).join('、')}`
                    : '内容无法识别'
                }}
              </n-text>
              <n-flex size="small" justify="end" wrap>
                <n-button
                  size="small"
                  secondary
                  :loading="downloadingName === item.name"
                  @click="emit('download', item)"
                >
                  下载
                </n-button>
                <n-button
                  size="small"
                  type="error"
                  secondary
                  :loading="deletingName === item.name"
                  :disabled="disabled"
                  @click="emit('delete', item)"
                >
                  删除
                </n-button>
              </n-flex>
            </li>
          </ul>
        </n-spin>
      </template>
    </ResponsiveDataView>
  </section>
</template>

<script setup lang="tsx">
import { computed } from 'vue';
import { filesize } from 'filesize';
import { NButton, NTag, type DataTableColumns } from 'naive-ui';
import type { FileItem } from '@/api';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';
import { describeBackupSelection } from '@/features/backup/viewModel';

const props = defineProps<{
  items: FileItem[];
  loading: boolean;
  downloadingName: string;
  deletingName: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  download: [item: FileItem];
  delete: [item: FileItem];
  openBatchDelete: [];
  openBackup: [];
}>();

const columns = computed<DataTableColumns<FileItem>>(() => [
  {
    title: '文件',
    key: 'name',
    minWidth: 280,
    render: row => {
      const desc = describeBackupSelection(row.selection);
      return (
        <div class="backup-file-list__file">
          <strong>{row.name}</strong>
          {desc.length > 0 ? (
            <span>包含：{desc.join('、')}</span>
          ) : (
            <span class="backup-file-list__unknown">内容无法识别</span>
          )}
        </div>
      );
    },
  },
  {
    title: '大小',
    key: 'fileSize',
    width: 110,
    render: row => (
      <NTag size="small" bordered={false}>
        {filesize(row.fileSize)}
      </NTag>
    ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    render: row => (
      <n-space justify="end">
        <NButton
          size="small"
          secondary
          loading={props.downloadingName === row.name}
          onClick={() => emit('download', row)}
        >
          下载
        </NButton>
        <NButton
          size="small"
          type="error"
          secondary
          loading={props.deletingName === row.name}
          disabled={props.disabled}
          onClick={() => emit('delete', row)}
        >
          删除
        </NButton>
      </n-space>
    ),
  },
]);
</script>

<style scoped>
.backup-file-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.backup-file-list__summary,
.backup-file-list__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.backup-file-list__summary {
  justify-content: space-between;
}

.backup-file-list__toolbar {
  flex-wrap: wrap;
  justify-content: flex-end;
}

:deep(.backup-file-list__file) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

:deep(.backup-file-list__file strong) {
  overflow-wrap: anywhere;
  font-weight: 650;
}

:deep(.backup-file-list__file span) {
  color: var(--sd-text-muted);
  font-size: 12px;
}

:deep(.backup-file-list__unknown) {
  color: var(--sd-warning);
}

.backup-file-list__compact-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.75rem;
  list-style: none;
}

.backup-file-list__compact-item {
  display: grid;
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
  gap: 0.65rem;
  padding: 0.75rem;
}

.backup-file-list__compact-heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.backup-file-list__compact-heading strong,
.backup-file-list__compact-selection {
  overflow-wrap: anywhere;
}

@media (max-width: 760px) {
  .backup-file-list__summary {
    align-items: flex-start;
    flex-direction: column;
  }

  .backup-file-list__toolbar {
    justify-content: flex-start;
  }
}
</style>
