<script setup lang="ts">
import { computed, ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { filesize } from 'filesize';
import { useDialog, useMessage } from 'naive-ui';
import {
  getSdApiV2StoryBackupDownload,
  getSdApiV2StoryBackupListOptions,
  postSdApiV2StoryBackupBatchDelete,
  type StoryLogBackup,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

const selectedBackups = ref<StoryLogBackup[]>([]);
const selectedBackupNames = computed({
  get: () => selectedBackups.value.map(item => item.name),
  set: names => {
    selectedBackups.value = backups.value.filter(item => names.includes(item.name));
  },
});
const checkAllBackups = ref(false);

const backupListQuery = useQuery({
  ...getSdApiV2StoryBackupListOptions(),
  enabled: hasAccessToken,
});

const backups = computed(() => backupListQuery.data.value?.item.data ?? []);
const isIndeterminate = computed(() => {
  return selectedBackups.value.length > 0 && selectedBackups.value.length < backups.value.length;
});
const selectedBytes = computed(() =>
  selectedBackups.value.map(item => item.fileSize).reduce((sum, size) => sum + size, 0),
);

const refreshList = () =>
  queryClient.invalidateQueries({
    queryKey: ['getSdApiV2StoryBackupList'],
  });

const deleteMutation = useMutation({
  mutationFn: async (names: string[]) => {
    const { data } = await postSdApiV2StoryBackupBatchDelete({
      body: {
        body: {
          names,
        },
      },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async item => {
    if (item.result) {
      message.success('已删除所选备份');
    } else {
      message.error('有备份删除失败！失败文件：\n' + (item.fails ?? []).join('\n'));
    }
    selectedBackups.value = [];
    checkAllBackups.value = false;
    await refreshList();
  },
  onError: () => {
    message.error('删除失败');
  },
});

function handleCheckAllChange(checked: boolean) {
  selectedBackups.value = checked ? [...backups.value] : [];
}

function handleCheckedBackupChange() {
  checkAllBackups.value = selectedBackups.value.length === backups.value.length;
}

function backupBatchDeleteConfirm() {
  dialog.warning({
    title: '提示',
    content: '确认删除选择的所有跑团日志备份？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(selectedBackups.value.map(item => item.name));
    },
  });
}

function bakDeleteConfirm(name: string) {
  dialog.warning({
    title: '提示',
    content: '确认删除？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync([name]);
    },
  });
}

async function downloadBackup(name: string) {
  const result = await getSdApiV2StoryBackupDownload({
    query: { name },
    parseAs: 'blob',
    throwOnError: true,
  });
  const blob = result.data as Blob;
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}
</script>

<template>
  <div class="tip">
    <n-text>
      每次向染色器上传跑团日志之前，都会在本地先保留一份备份，再进行上传。<br />
      确定不再需要时，你可以在此处删除这些备份文件。<br /><br />
      <strong>删除此处的备份文件不会使日志丢失。</strong>
    </n-text>
  </div>

  <header class="backup-header">
    <n-flex size="large" align="center">
      <n-checkbox
        v-model:checked="checkAllBackups"
        :indeterminate="isIndeterminate"
        :disabled="!(backups && backups.length > 0)"
        @update:checked="handleCheckAllChange"
      >
        {{ checkAllBackups ? '取消全选' : '全选' }}
      </n-checkbox>
      <n-text type="info" class="text-xs">
        已勾选 {{ selectedBackups.length }} 个备份，共 {{ filesize(selectedBytes) }}
      </n-text>
    </n-flex>

    <n-button
      type="error"
      :disabled="!(selectedBackups && selectedBackups.length > 0)"
      :loading="deleteMutation.isPending.value"
      @click="backupBatchDeleteConfirm"
    >
      删除所选
    </n-button>
  </header>

  <main class="backup-list">
    <n-checkbox-group v-model:value="selectedBackupNames" @update:value="handleCheckedBackupChange">
      <div v-for="backup in backups" :key="backup.name" class="backup-line">
        <n-checkbox :value="backup.name" :label="backup.name" />
        <n-flex size="small" wrap class="backup-actions">
          <n-button size="small" secondary @click="downloadBackup(backup.name)">
            下载 - {{ filesize(backup.fileSize) }}
          </n-button>
          <n-button type="error" size="small" secondary @click="bakDeleteConfirm(backup.name)">
            删除
          </n-button>
        </n-flex>
      </div>
    </n-checkbox-group>
  </main>
</template>

<style scoped>
.backup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 1rem 0;
  padding: 0 1rem;
  gap: 1rem;
}

.backup-list {
  display: flex;
  flex-direction: column;
}

.backup-line {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
  padding: 0.4rem 1rem;
}

.backup-line:not(:first-child) {
  border-top: 1px solid var(--sd-border-soft);
}

.backup-actions {
  margin-left: auto;
}

@media screen and (max-width: 700px) {
  .backup-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .backup-actions {
    margin-left: 0;
  }
}
</style>
