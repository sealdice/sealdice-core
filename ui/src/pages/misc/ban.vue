<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useMutation, useQuery } from '@tanstack/vue-query';
import { isEqual } from 'es-toolkit/compat';
import {
  getSdApiV2BanConfig,
  getSdApiV2BanExport,
  postSdApiV2BanAdd,
  postSdApiV2BanDelete,
  postSdApiV2BanImport,
  postSdApiV2BanList,
  putSdApiV2BanConfig,
  type BanConfig,
  type BanListInfoItem,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import BanAddDialog from '@/components/ban/BanAddDialog.vue';
import BanConfigPanel from '@/components/ban/BanConfigPanel.vue';
import BanListPanel from '@/components/ban/BanListPanel.vue';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { useUnsavedChanges } from '@/features/unsavedChanges';
import {
  buildBanAddPayload,
  buildBanListPayload,
  createDefaultBanAddForm,
  createDefaultBanListQuery,
  isBanImportFileAccepted,
  normalizeBanConfig,
} from '@/features/ban/viewModel';

const message = useMessage();
const dialog = useDialog();

const tab = ref<'list' | 'config'>('list');
const listQuery = reactive(createDefaultBanListQuery());
const addDialogVisible = ref(false);
const addForm = ref(createDefaultBanAddForm());
const configDraft = ref<BanConfig | null>(null);
const initialConfig = ref<BanConfig | null>(null);

const listParams = computed(() => buildBanListPayload(listQuery));
const configDirty = computed(() => Boolean(configDraft.value && initialConfig.value && !isEqual(configDraft.value, initialConfig.value)));

const listQueryResult = useQuery({
  queryKey: computed(() => ['ban-list', listParams.value]),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await postSdApiV2BanList({
      body: listParams.value,
      throwOnError: true,
    });
    return data.item;
  },
});

const configQuery = useQuery({
  queryKey: ['ban-config'],
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2BanConfig({
      throwOnError: true,
    });
    return normalizeBanConfig(data.item);
  },
});

watch(
  () => configQuery.data.value,
  value => {
    if (!value) return;
    configDraft.value = structuredClone(value);
    initialConfig.value = structuredClone(value);
  },
  { immediate: true },
);

const addMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2BanAdd({
      body: buildBanAddPayload(addForm.value),
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async () => {
    message.success('已保存');
    addDialogVisible.value = false;
    addForm.value = createDefaultBanAddForm();
    await listQueryResult.refetch();
  },
});

const deleteMutation = useMutation({
  mutationFn: async (item: BanListInfoItem) => {
    await postSdApiV2BanDelete({
      body: {
        id: item.ID,
      },
      throwOnError: true,
    });
  },
  onSuccess: async () => {
    message.success('已删除');
    await listQueryResult.refetch();
  },
});

const importMutation = useMutation({
  mutationFn: async (file: File) => {
    const { data } = await postSdApiV2BanImport({
      body: { file },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async () => {
    message.success('导入黑白名单完成');
    await listQueryResult.refetch();
  },
});

const saveConfigMutation = useMutation({
  mutationFn: async (config: BanConfig) => {
    const { data } = await putSdApiV2BanConfig({
      body: config,
      throwOnError: true,
    });
    return normalizeBanConfig(data.item);
  },
  onSuccess: (config) => {
    configDraft.value = structuredClone(config);
    initialConfig.value = structuredClone(config);
    message.success('已保存');
  },
});

useUnsavedChanges('ban-config', {
  label: '拉黑设置',
  dirty: configDirty,
  save: async () => {
    if (!configDraft.value) return;
    await saveConfigMutation.mutateAsync(configDraft.value);
  },
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: computed(() => Boolean(configDraft.value) && configDirty.value),
  confirmMessage: '拉黑设置还有修改，确定要忽略？',
});

const listItems = computed(() => listQueryResult.data.value?.list ?? []);
const listTotal = computed(() => Number(listQueryResult.data.value?.total ?? 0));

function updateListQuery(patch: Partial<typeof listQuery>) {
  Object.assign(listQuery, patch);
}

function openAddDialog() {
  addForm.value = createDefaultBanAddForm();
  addDialogVisible.value = true;
}

function confirmDelete(item: BanListInfoItem) {
  dialog.warning({
    title: '删除',
    content: '是否删除此记录？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(item);
    },
  });
}

async function importFile(file: File) {
  if (!isBanImportFileAccepted(file.name)) {
    message.error('仅支持导入 .json 黑白名单文件');
    return;
  }
  try {
    await importMutation.mutateAsync(file);
  } catch (error) {
    message.error(getErrorMessage(error, '导入黑白名单失败'));
  }
}

async function exportFile() {
  try {
    await downloadApiFile(
      getSdApiV2BanExport({
        responseType: 'blob',
        throwOnError: true,
      }),
      '黑白名单.json',
    );
  } catch (error) {
    message.error(getErrorMessage(error, '导出黑白名单失败'));
  }
}

async function saveConfig() {
  if (!configDraft.value) return;
  await saveConfigMutation.mutateAsync(configDraft.value);
}
</script>

<template>
  <main class="ban-page">
    <header class="ban-page__header">
      <div>
        <h1>黑白名单</h1>
        <p>管理黑白名单条目与拉黑惩罚策略，当前页面已迁移至 V2 API。</p>
      </div>
    </header>

    <n-tabs v-model:value="tab" animated>
      <n-tab-pane name="list" tab="黑白名单">
        <BanListPanel
          :items="listItems"
          :total="listTotal"
          :query="listQuery"
          :loading="listQueryResult.isFetching.value"
          :add-pending="addMutation.isPending.value"
          :import-pending="importMutation.isPending.value"
          @update-query="updateListQuery"
          @open-add="openAddDialog"
          @delete="confirmDelete"
          @import="importFile"
          @export="exportFile"
        />
      </n-tab-pane>
      <n-tab-pane name="config" tab="拉黑设置">
        <BanConfigPanel
          v-if="configDraft"
          v-model:config="configDraft"
          :dirty="configDirty"
          :saving="saveConfigMutation.isPending.value"
          @save="saveConfig"
        />
      </n-tab-pane>
    </n-tabs>

    <BanAddDialog
      v-model:show="addDialogVisible"
      v-model:form="addForm"
      :submitting="addMutation.isPending.value"
      @submit="addMutation.mutate()"
    />
  </main>
</template>

<style scoped>
.ban-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.ban-page__header h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1.75rem;
}

.ban-page__header p {
  margin: 0.5rem 0 0;
  color: var(--sd-text-secondary);
}
</style>
