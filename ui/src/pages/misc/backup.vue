<template>
  <main class="backup-page">
    <header class="page-head">
      <div class="page-head-copy">
        <h1>备份</h1>
        <p>管理自动备份策略与历史备份文件。</p>
      </div>
    </header>

    <n-alert v-if="configErrorText" type="error" :bordered="false">
      {{ configErrorText }}
    </n-alert>
    <n-alert v-if="listErrorText" type="error" :bordered="false">
      {{ listErrorText }}
    </n-alert>

    <n-tabs v-model:value="activeTab" type="line" animated class="backup-tabs">
      <n-tab-pane name="settings" tab="备份设置">
        <n-spin :show="configQuery.isLoading.value && !configDraft">
          <BackupConfigPanel
            v-if="configDraft"
            v-model:config="configDraft"
            :dirty="configDirty"
            :saving="saveConfigMutation.isPending.value"
            :timestamp="timestamp"
            :disabled="isTestMode"
            @save="saveConfig"
          />
        </n-spin>
      </n-tab-pane>

      <n-tab-pane name="files" tab="备份文件">
        <BackupFileList
          :items="items"
          :loading="listQuery.isFetching.value"
          :downloading-name="downloadingName"
          :deleting-name="deletingName"
          :disabled="isTestMode"
          @download="downloadBackup"
          @delete="confirmDelete"
          @open-batch-delete="openBatchDeleteDialog"
          @open-backup="openBackupDialog"
        />
      </n-tab-pane>
    </n-tabs>

    <BackupExecDialog
      v-model:show="execDialogVisible"
      v-model:selections="execSelections"
      :timestamp="timestamp"
      :pending="execMutation.isPending.value"
      @submit="executeBackup"
    />

    <BackupBatchDeleteDialog
      v-model:show="batchDeleteVisible"
      v-model:selected-names="batchDeleteNames"
      :items="items"
      :pending="batchDeleteMutation.isPending.value"
      @submit="confirmBatchDelete"
    />
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
import { useMutation, useQuery } from '@tanstack/vue-query';
import { isEqual } from 'es-toolkit/compat';
import dayjs from 'dayjs';
import {
  getSdApiV2BackupConfigOptions,
  getSdApiV2BackupDownload,
  getSdApiV2BackupListOptions,
  postSdApiV2BackupBatchDelete,
  postSdApiV2BackupConfigSave,
  postSdApiV2BackupDelete,
  postSdApiV2BackupExec,
  type ConfigWritable,
  type FileItem,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import BackupBatchDeleteDialog from '@/components/backup/BackupBatchDeleteDialog.vue';
import BackupConfigPanel from '@/components/backup/BackupConfigPanel.vue';
import BackupExecDialog from '@/components/backup/BackupExecDialog.vue';
import BackupFileList from '@/components/backup/BackupFileList.vue';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import {
  buildBackupConfigPayload,
  formatBackupSelection,
  getDefaultBatchDeleteNames,
  normalizeBackupConfig,
  parseBackupSelection,
  type BackupConfigDraft,
  type BackupSelectionKey,
} from '@/features/backup/viewModel';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  useTestMode,
} from '@/features/testMode/state';
import { useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const dialog = useDialog();
const { isTestMode } = useTestMode();

const timestamp = shallowRef(dayjs().format('YYMMDD_HHmmss'));
const activeTab = shallowRef<'settings' | 'files'>('settings');
const configDraft = ref<BackupConfigDraft | null>(null);
const configBaseline = ref<ConfigWritable | null>(null);
const execDialogVisible = shallowRef(false);
const batchDeleteVisible = shallowRef(false);
const execSelections = ref<BackupSelectionKey[]>(parseBackupSelection(0b111111));
const batchDeleteNames = ref<string[]>([]);
const downloadingName = shallowRef('');
const deletingName = shallowRef('');

let timerId: ReturnType<typeof setInterval> | null = null;

const configQuery = useQuery({
  ...getSdApiV2BackupConfigOptions(),
  enabled: hasAccessToken,
});

const listQuery = useQuery({
  ...getSdApiV2BackupListOptions(),
  enabled: hasAccessToken,
});

const items = computed<FileItem[]>(() => listQuery.data.value?.item.items ?? []);
const configDirty = computed(() =>
  Boolean(
    configDraft.value &&
      configBaseline.value &&
      !isEqual(buildBackupConfigPayload(configDraft.value), configBaseline.value)
  )
);
const configErrorText = computed(() =>
  configQuery.isError.value ? getErrorMessage(configQuery.error.value, '读取备份设置失败') : ''
);
const listErrorText = computed(() =>
  listQuery.isError.value ? getErrorMessage(listQuery.error.value, '读取备份文件列表失败') : ''
);

const saveConfigMutation = useMutation({
  mutationFn: async (config: BackupConfigDraft) => {
    const payload = buildBackupConfigPayload(config);
    await postSdApiV2BackupConfigSave({
      body: payload,
      throwOnError: true,
    });
    return payload;
  },
  onSuccess: async payload => {
    configBaseline.value = structuredClone(payload);
    message.success('已保存');
    await configQuery.refetch();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, '保存备份设置失败'));
  },
});

const execMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2BackupExec({
      body: {
        selection: formatBackupSelection(execSelections.value),
      },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async result => {
    execDialogVisible.value = false;
    if (!result.success) {
      message.error('备份失败');
      return;
    }
    message.success('已进行备份');
    await listQuery.refetch();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, '备份失败'));
  },
});

const deleteMutation = useMutation({
  mutationFn: async (name: string) => {
    deletingName.value = name;
    const { data } = await postSdApiV2BackupDelete({
      body: { name },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async result => {
    if (!result.success) {
      message.error('删除失败');
      return;
    }
    message.success('已删除');
    await listQuery.refetch();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, '删除失败'));
  },
  onSettled: () => {
    deletingName.value = '';
  },
});

const batchDeleteMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2BackupBatchDelete({
      body: { names: batchDeleteNames.value },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async result => {
    const fails = result.fails ?? [];
    if (fails.length > 0) {
      message.error(`有备份删除失败：${fails.join('、')}`);
      return;
    }
    batchDeleteVisible.value = false;
    batchDeleteNames.value = [];
    message.success('已删除所选备份');
    await listQuery.refetch();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, '删除备份失败'));
  },
});

watch(
  () => configQuery.data.value?.item,
  value => {
    if (!value) return;
    const next = normalizeBackupConfig(value);
    configDraft.value = structuredClone(next);
    configBaseline.value = buildBackupConfigPayload(next);
  },
  { immediate: true }
);

useUnsavedChanges('backup-config', {
  label: '备份设置',
  dirty: configDirty,
  save: async () => {
    if (!configDraft.value) return;
    await saveConfigMutation.mutateAsync(configDraft.value);
  },
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: computed(() => Boolean(configDraft.value) && configDirty.value),
  confirmMessage: '备份设置还有修改，确定要忽略？',
});

function openBackupDialog() {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  execSelections.value = parseBackupSelection(0b111111);
  execDialogVisible.value = true;
}

function openBatchDeleteDialog() {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  batchDeleteNames.value = getDefaultBatchDeleteNames(items.value);
  batchDeleteVisible.value = true;
}

async function saveConfig() {
  if (!configDraft.value) return;
  try {
    await saveConfigMutation.mutateAsync(configDraft.value);
  } catch (error) {
    message.error(getErrorMessage(error, '保存备份设置失败'));
  }
}

async function downloadBackup(item: FileItem) {
  downloadingName.value = item.name;
  try {
    await downloadApiFile(
      getSdApiV2BackupDownload({
        query: { name: item.name },
        responseType: 'blob',
        throwOnError: true,
      }),
      item.name
    );
  } catch (error) {
    message.error(getErrorMessage(error, '下载备份失败'));
  } finally {
    downloadingName.value = '';
  }
}

function confirmDelete(item: FileItem) {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '删除备份',
    content: `确认删除「${item.name}」？`,
    positiveText: '删除',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(item.name);
    },
  });
}

function confirmBatchDelete() {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '批量删除备份',
    content: '确认删除所选备份？删除的内容无法找回。',
    positiveText: '删除所选',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      await batchDeleteMutation.mutateAsync();
    },
  });
}

function executeBackup() {
  execMutation.mutate();
}

onMounted(() => {
  timerId = setInterval(() => {
    timestamp.value = dayjs().format('YYMMDD_HHmmss');
  }, 1000);
});

onBeforeUnmount(() => {
  if (timerId) clearInterval(timerId);
});
</script>

<style scoped>
.backup-page {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.page-head-copy h1 {
  margin: 0;
  font-size: 1.6rem;
}

.page-head-copy p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
}

.backup-tabs {
  min-width: 0;
}

@media (max-width: 639.9px) {
  .page-head {
    gap: 0.65rem;
  }
}
</style>
