<template>
  <main class="helpdoc-page">
    <header class="page-header">
      <n-button
        type="primary"
        :loading="reloadMutation.isPending.value"
        :disabled="reloadMutation.isPending.value"
        @click="reloadMutation.mutate()"
      >
        <template #icon>
          <n-icon><i-carbon-renew /></n-icon>
        </template>
        重载帮助文档
      </n-button>
    </header>

    <n-affix v-if="needReload" :top="60">
      <TipBox type="error">
        <n-text type="error" class="text-base" tag="strong">存在修改，需要重载后生效！</n-text>
      </TipBox>
    </n-affix>

    <n-tabs v-model:value="tab" justify-content="space-evenly" class="helpdoc-tabs">
      <n-tab-pane tab="文件" name="file">
        <HelpdocFilePane
          v-model:checked-keys="checkedFileKeys"
          :doc-tree="docTree"
          :loading="treeQuery.isFetching.value"
          :active-upload-tasks="activeUploadTasks"
          :deleting="deleteMutation.isPending.value"
          @open-upload="uploadDialogVisible = true"
          @open-config="configDialogVisible = true"
          @delete-files="deleteFiles"
          @retry-task="retryTask"
        />
      </n-tab-pane>

      <n-tab-pane tab="词条" name="item">
        <HelpdocItemPane
          v-model:query="itemQuery"
          :loading="pageBusy"
          :items="helpItems"
          :total="itemTotal"
          :group-options="itemGroupOptions"
          :columns="columns"
          @search="queryItems"
          @reset="resetItemQuery"
        />
      </n-tab-pane>
    </n-tabs>

    <HelpdocUploadDialog
      v-model:show="uploadDialogVisible"
      v-model:group="uploadGroup"
      v-model:file-list="uploadFiles"
      :groups="docGroups"
      :busy="uploader.busy.value"
      @submit="submitUpload"
    />

    <HelpdocConfigDialog
      v-model:show="configDialogVisible"
      :groups="docGroups"
      :aliases="currentAliases"
      :saving="saveConfigMutation.isPending.value"
      @save="saveConfig"
      @add-alias="addAlias"
      @remove-alias="removeAlias"
    />
  </main>
</template>

<script setup lang="tsx">
import { computed, h, onMounted, reactive, ref, shallowRef, watch, type CSSProperties } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';
import {
  NTag,
  NText,
  NTooltip,
  type DataTableColumns,
  type UploadFileInfo,
} from 'naive-ui';
import type { HelpTextVo } from '@/api';
import HelpdocConfigDialog from '@/components/helpdoc/HelpdocConfigDialog.vue';
import HelpdocFilePane from '@/components/helpdoc/HelpdocFilePane.vue';
import HelpdocItemPane from '@/components/helpdoc/HelpdocItemPane.vue';
import HelpdocUploadDialog from '@/components/helpdoc/HelpdocUploadDialog.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { useHelpdocConfigDraft } from '@/features/helpdoc/configDraft';
import { useHelpdocMutations } from '@/features/helpdoc/mutations';
import {
  createDefaultHelpdocItemQuery,
  type HelpdocItemQueryModel,
  useHelpdocQueries,
} from '@/features/helpdoc/queries';
import { useHelpdocUpload } from '@/features/helpdoc/upload';
import {
  getHelpdocTextPreview,
  getHelpdocTextTooltip,
} from '@/features/helpdoc/viewModel';
import { useUnsavedChanges } from '@/features/unsavedChanges';
import type { ResumableUploadTask } from '@/features/upload/resumableUpload';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

const tab = shallowRef<'file' | 'item'>('file');
const needReload = ref(false);
const uploadDialogVisible = ref(false);
const configDialogVisible = ref(false);
const checkedFileKeys = ref<Array<string | number>>([]);
const uploadFiles = ref<UploadFileInfo[]>([]);
const uploadGroup = ref('');
const itemQuery = reactive(createDefaultHelpdocItemQuery());
const appliedItemQuery = ref<HelpdocItemQueryModel>({ ...itemQuery });

const {
  treeQuery,
  configQuery,
  itemsQuery,
  docTree,
  docGroups,
  itemGroupOptions,
  helpItems,
  itemTotal,
} = useHelpdocQueries(appliedItemQuery);

const configDraft = useHelpdocConfigDraft();

watch(
  () => configQuery.data.value,
  aliases => {
    configDraft.syncRemote(aliases ?? {});
  },
  { immediate: true },
);

watch(
  () => [itemQuery.pageNum, itemQuery.pageSize] as const,
  () => {
    appliedItemQuery.value = { ...itemQuery };
  },
);

watch(uploadDialogVisible, visible => {
  if (visible) return;
  uploadFiles.value = [];
  uploadGroup.value = '';
});

const pageBusy = computed(() => treeQuery.isFetching.value || itemsQuery.isFetching.value);
const currentAliases = computed(() => configDraft.currentAliases.value);
const helpContentTooltipStyle: CSSProperties = {
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  overflowWrap: 'anywhere',
  maxWidth: '520px',
};

const uploader = useHelpdocUpload({
  group: () => uploadGroup.value,
  async onSuccess(task: ResumableUploadTask) {
    message.success(`上传完成：${task.filename}`);
    needReload.value = true;
    await queryClient.invalidateQueries({ queryKey: ['helpdoc-tree'] });
  },
  onError(task: ResumableUploadTask) {
    message.error(`上传失败：${task.filename}`);
  },
});

const activeUploadTasks = computed(() =>
  uploader.tasks.value.filter(task => task.status !== 'success'),
);

const { reloadMutation, deleteMutation, saveConfigMutation } = useHelpdocMutations({
  queryClient,
  message,
  getConfigPayload: () => configDraft.payload.value,
  onReloaded: () => {
    needReload.value = false;
  },
  onDeleted: () => {
    needReload.value = true;
    checkedFileKeys.value = [];
  },
  onConfigSaved: () => {
    configDraft.commitSaved();
    configDialogVisible.value = false;
  },
});

useUnsavedChanges('helpdoc-config', {
  label: '帮助文档设置',
  dirty: computed(() => configDraft.dirty.value),
  save: saveConfig,
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: computed(() => configDraft.dirty.value),
  confirmMessage: '帮助文档设置还有修改，确定要忽略？',
});

const columns: DataTableColumns<HelpTextVo> = [
  { title: '序号', key: 'id', align: 'center' },
  {
    title: '分组',
    key: 'group',
    align: 'center',
    render: row =>
      h(
        NTag,
        { type: row.group === 'builtin' ? 'info' : 'success', size: 'small', bordered: false },
        { default: () => row.group || '-' },
      ),
  },
  { title: '来源文件', key: 'from', minWidth: 180, ellipsis: { tooltip: true } },
  { title: '词条名', key: 'title', minWidth: 140, ellipsis: { tooltip: true } },
  {
    title: '内容',
    key: 'content',
    minWidth: 320,
    render: row =>
      h(
        NTooltip,
        {
          trigger: 'hover',
          width: 520,
          contentStyle: helpContentTooltipStyle,
        },
        {
          trigger: () =>
            h(NText, { class: 'help-content-preview' }, { default: () => getHelpdocTextPreview(row.content ?? '') }),
          default: () => getHelpdocTextTooltip(row.content ?? ''),
        },
      ),
  },
  { title: '分类', key: 'packageName', minWidth: 130, ellipsis: { tooltip: true } },
];

onMounted(() => {
  uploader.restore();
});

function queryItems() {
  itemQuery.pageNum = 1;
  appliedItemQuery.value = { ...itemQuery };
  void itemsQuery.refetch();
}

function resetItemQuery() {
  Object.assign(itemQuery, createDefaultHelpdocItemQuery());
  appliedItemQuery.value = { ...itemQuery };
  void itemsQuery.refetch();
}

async function submitUpload(files: File[]) {
  await uploader.enqueueFiles(files);
  uploadDialogVisible.value = false;
  uploadFiles.value = [];
}

function deleteFiles() {
  const keys = checkedFileKeys.value.map(String);
  if (!keys.length) {
    message.error('未选择文件');
    return;
  }
  dialog.warning({
    title: '删除',
    content: '确认删除选择的文件吗？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(keys);
    },
  });
}

function addAlias(groupKey: string, alias: string) {
  if (configDraft.addAlias(groupKey, alias)) return;
  if (alias.trim()) {
    message.error(`别名 ${alias.trim()} 已被使用`);
  }
}

function removeAlias(groupKey: string, alias: string) {
  configDraft.removeAlias(groupKey, alias);
}

async function saveConfig() {
  await saveConfigMutation.mutateAsync();
}

function retryTask(task: ResumableUploadTask) {
  void uploader.retry(task);
}
</script>

<style scoped>
.helpdoc-page {
  width: 100%;
}

.page-header {
  margin-bottom: 1rem;
}

.helpdoc-tabs {
  padding-bottom: 2rem;
}

.helpdoc-tabs :deep(.n-tabs-nav-scroll-content) {
  min-width: max-content;
}

:deep(.help-content-preview) {
  display: inline-block;
  max-width: 30rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

@media screen and (max-width: 639.9px) {
  .helpdoc-tabs :deep(.n-tabs-nav-scroll-content) {
    justify-content: flex-start !important;
  }

  .page-header {
    display: flex;
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
