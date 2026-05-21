<script setup lang="tsx">
import { computed } from 'vue';
import { filesize } from 'filesize';
import { NButton, NTag, type DataTableColumns } from 'naive-ui';
import type { FileItem } from '@/api';
import { describeBackupSelection } from '@/features/backup/viewModel';

const props = defineProps<{
  items: FileItem[];
  loading: boolean;
  downloadingName: string;
  deletingName: string;
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
        <div class='backup-file-list__file'>
          <strong>{row.name}</strong>
          {desc.length > 0 ? (
            <span>包含：{desc.join('、')}</span>
          ) : (
            <span class='backup-file-list__unknown'>内容无法识别</span>
          )}
        </div>
      );
    },
  },
  {
    title: '大小',
    key: 'fileSize',
    width: 110,
    render: row => <NTag size='small' bordered={false}>{filesize(row.fileSize)}</NTag>,
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    render: row => (
      <n-space justify='end'>
        <NButton
          size='small'
          secondary
          loading={props.downloadingName === row.name}
          onClick={() => emit('download', row)}
        >
          下载
        </NButton>
        <NButton
          size='small'
          type='error'
          secondary
          loading={props.deletingName === row.name}
          onClick={() => emit('delete', row)}
        >
          删除
        </NButton>
      </n-space>
    ),
  },
]);
</script>

<template>
  <n-card class="backup-file-list" :bordered="false">
    <template #header>
      <div class="backup-file-list__head">
        <div>
          <h2>已备份文件</h2>
          <p>{{ items.length }} 个备份文件</p>
        </div>
        <n-space>
          <n-button type="primary" @click="emit('openBackup')">
            立即备份
          </n-button>
          <n-button type="error" secondary :disabled="items.length === 0" @click="emit('openBatchDelete')">
            <template #icon>
              <n-icon>
                <i-carbon-row-delete />
              </n-icon>
            </template>
            批量删除
          </n-button>
        </n-space>
      </div>
    </template>

    <n-empty v-if="!loading && items.length === 0" description="暂无备份文件">
      <template #extra>
        <n-button type="primary" @click="emit('openBackup')">
          立即备份
        </n-button>
      </template>
    </n-empty>

    <n-data-table
      v-else
      :columns="columns"
      :data="items"
      :loading="loading"
      :bordered="false"
      :row-key="row => row.name"
      :scroll-x="680"
      size="small"
    />
  </n-card>
</template>

<style scoped>
.backup-file-list__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.backup-file-list__head h2 {
  margin: 0;
}

.backup-file-list__head p {
  margin: 4px 0 0;
  color: var(--sd-text-muted);
  font-size: 13px;
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

@media (max-width: 760px) {
  .backup-file-list__head {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
