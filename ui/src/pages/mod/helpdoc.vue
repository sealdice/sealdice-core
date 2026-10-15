<template>
  <main class="helpdoc-page sd-page-flow">
    <PageHeader title="帮助文档">
      <n-button
        v-if="!needReload"
        type="primary"
        :loading="reloadMutation.isPending.value"
        :disabled="reloadMutation.isPending.value"
        @click="reloadMutation.mutate()"
      >
        <template #icon>
          <n-icon><i-tabler-refresh /></n-icon>
        </template>
        重载帮助文档
      </n-button>
    </PageHeader>

    <n-tabs v-model:value="tab" class="helpdoc-tabs sd-scrollable-tabs">
      <n-tab-pane tab="文件" name="file">
        <HelpdocFilePane
          v-model:checked-keys="checkedFileKeys"
          :doc-tree="docTree"
          :loading="treeQuery.isFetching.value"
          :deleting="deleteMutation.isPending.value"
          @open-upload="uploadDialogVisible = true"
          @open-config="configDialogVisible = true"
          @delete-files="deleteFiles"
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
      :busy="helpdocUploading"
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
import { computed, h, ref, shallowRef, watch, type CSSProperties } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { NTag, NText, NTooltip, type DataTableColumns, type UploadFileInfo } from 'naive-ui';
import {
  getSdApiV2HelpdocTreeOptions,
  getSdApiV2HelpdocTreeQueryKey,
  getSdApiV2HelpdocConfigOptions,
  getSdApiV2HelpdocConfigQueryKey,
  getSdApiV2HelpdocItemsPageOptions,
  getSdApiV2HelpdocItemsPageQueryKey,
  postSdApiV2HelpdocConfig,
  postSdApiV2HelpdocDelete,
  postSdApiV2HelpdocReload,
  type HelpTextVo,
  type GetSdApiV2HelpdocItemsPageResponse,
} from '@/api';
import HelpdocConfigDialog from '@/components/helpdoc/HelpdocConfigDialog.vue';
import HelpdocFilePane from '@/components/helpdoc/HelpdocFilePane.vue';
import HelpdocItemPane from '@/components/helpdoc/HelpdocItemPane.vue';
import HelpdocUploadDialog from '@/components/helpdoc/HelpdocUploadDialog.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import {
  createDefaultHelpdocItemQuery,
  type HelpdocItemQueryModel,
  buildHelpdocGroupOptions,
  buildHelpdocItemParams,
  convertHelpdocTree,
  getHelpdocItemGroupOptions,
  buildHelpdocConfigPayload,
  cloneHelpdocAliases,
  isHelpdocConfigDirty,
  normalizeHelpdocAliases,
  getHelpdocTextPreview,
  getHelpdocTextTooltip,
  summarizeHelpdocDelete,
} from '@/features/helpdoc/viewModel';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
} from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';
import { useUploadStore } from '@/features/upload/store';
import { usePendingReload, useUnsavedChanges } from '@/features/unsavedChanges';

const uploadStore = useUploadStore();
const { enqueue: enqueueUploads } = uploadStore;
const { tasks: uploadTasks } = storeToRefs(uploadStore);

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const { hasAccessToken } = storeToRefs(useAuthStore());

const tab = shallowRef<'file' | 'item'>('file');
const needReload = ref(false);
const uploadDialogVisible = ref(false);
const configDialogVisible = ref(false);
const checkedFileKeys = ref<Array<string | number>>([]);
const uploadFiles = ref<UploadFileInfo[]>([]);
const uploadGroup = ref('');
const itemQuery = ref(createDefaultHelpdocItemQuery());
const appliedItemQuery = ref<HelpdocItemQueryModel>({ ...itemQuery.value });

const treeQuery = useQuery({
  ...getSdApiV2HelpdocTreeOptions(),
  enabled: hasAccessToken,
  select: data => data.item.data ?? [],
});
const configQuery = useQuery({
  ...getSdApiV2HelpdocConfigOptions(),
  enabled: hasAccessToken,
  select: data => data.item.aliases ?? {},
});
const itemsQuery = useQuery(
  computed(() => ({
    ...getSdApiV2HelpdocItemsPageOptions({ query: buildHelpdocItemParams(appliedItemQuery.value) }),
    enabled: hasAccessToken.value,
    select: (data: GetSdApiV2HelpdocItemsPageResponse) => data.item,
  }))
);

const rawTree = computed(() => treeQuery.data.value ?? []);
const docTree = computed(() => rawTree.value.map(convertHelpdocTree));
const docGroups = computed(() => buildHelpdocGroupOptions(rawTree.value));
const itemGroupOptions = computed(() => getHelpdocItemGroupOptions(rawTree.value));
const helpItems = computed(() => itemsQuery.data.value?.data ?? []);
const itemTotal = computed(() => itemsQuery.data.value?.total ?? 0);

const currentAliases = ref(new Map<string, string[]>());
const initialAliases = ref(new Map<string, string[]>());
const configDirty = computed(() =>
  isHelpdocConfigDirty(currentAliases.value, initialAliases.value)
);
let configReady = false;
watch(
  configQuery.data,
  aliases => {
    if (configReady && configDirty.value) return;
    const normalized = normalizeHelpdocAliases(aliases ?? {});
    currentAliases.value = cloneHelpdocAliases(normalized);
    initialAliases.value = cloneHelpdocAliases(normalized);
    configReady = true;
  },
  { immediate: true }
);

watch(
  () => [itemQuery.value.pageNum, itemQuery.value.pageSize] as const,
  () => {
    appliedItemQuery.value = { ...itemQuery.value };
  }
);

watch(uploadDialogVisible, visible => {
  if (visible) return;
  uploadFiles.value = [];
  uploadGroup.value = '';
});

const pageBusy = computed(() => treeQuery.isFetching.value || itemsQuery.isFetching.value);
const helpContentTooltipStyle: CSSProperties = {
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  overflowWrap: 'anywhere',
  maxWidth: '520px',
};

const helpdocUploading = computed(() =>
  uploadTasks.value.some(
    task =>
      task.kind === 'helpdoc' &&
      (task.status === 'queued' || task.status === 'hashing' || task.status === 'uploading')
  )
);

const reloadMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2HelpdocReload({ throwOnError: true });
    return data.item;
  },
  onSuccess: async item => {
    if (isTestModeResponse(item)) {
      message.warning('展示模式无法重载帮助文档');
      return;
    }
    message.success('重载帮助文件成功');
    needReload.value = false;
    await Promise.all(
      [
        getSdApiV2HelpdocTreeQueryKey(),
        getSdApiV2HelpdocItemsPageQueryKey(),
        getSdApiV2HelpdocConfigQueryKey(),
      ].map(queryKey => queryClient.invalidateQueries({ queryKey }))
    );
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    notifyError(error, message, '重载帮助文件失败');
  },
});
const deleteMutation = useMutation({
  mutationFn: async (keys: string[]) => {
    const { data } = await postSdApiV2HelpdocDelete({ body: { keys }, throwOnError: true });
    return data.item;
  },
  onSuccess: async item => {
    if (!item.success) {
      message.error(item.err || '删除文件失败');
      return;
    }
    message.success('删除文件成功');
    needReload.value = true;
    checkedFileKeys.value = [];
    await queryClient.invalidateQueries({ queryKey: getSdApiV2HelpdocTreeQueryKey() });
  },
  onError: error => notifyError(error, message, '删除文件失败'),
});
const saveConfigMutation = useMutation({
  mutationFn: async () => {
    const snapshot = buildHelpdocConfigPayload(cloneHelpdocAliases(currentAliases.value));
    const { data } = await postSdApiV2HelpdocConfig({
      body: { aliases: snapshot },
      throwOnError: true,
    });
    return { item: data.item, snapshot };
  },
  onSuccess: async ({ item, snapshot }) => {
    if (!item.success) {
      message.error(item.err || '保存设置失败');
      return;
    }
    message.success('保存设置成功');
    initialAliases.value = normalizeHelpdocAliases(snapshot);
    configDialogVisible.value = false;
    await queryClient.invalidateQueries({ queryKey: getSdApiV2HelpdocConfigQueryKey() });
  },
  onError: error => notifyError(error, message, '保存设置失败'),
});

useUnsavedChanges('helpdoc-config', {
  label: '帮助文档设置',
  dirty: configDirty,
  save: saveConfig,
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: configDirty,
  discard: () => {
    currentAliases.value = cloneHelpdocAliases(initialAliases.value);
  },
  confirmMessage: '帮助文档设置还有修改，确定要忽略？',
});

usePendingReload('helpdoc-reload', {
  label: '帮助文档',
  pending: needReload,
  reload: () => reloadMutation.mutateAsync(),
  reloading: computed(() => reloadMutation.isPending.value),
  canReload: computed(() => !reloadMutation.isPending.value),
  actionText: '重载帮助文档',
});

const columns: DataTableColumns<HelpTextVo> = [
  { title: '序号', key: 'id', width: 80, align: 'center' },
  {
    title: '分组',
    key: 'group',
    width: 120,
    align: 'center',
    render: row =>
      h(
        NTag,
        { type: row.group === 'builtin' ? 'info' : 'success', size: 'small', bordered: false },
        { default: () => row.group || '-' }
      ),
  },
  { title: '来源文件', key: 'from', width: 200, ellipsis: { tooltip: true } },
  { title: '词条名', key: 'title', width: 180, ellipsis: { tooltip: true } },
  {
    title: '内容',
    key: 'content',
    width: 480,
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
            h(
              NText,
              { class: 'help-content-preview' },
              { default: () => getHelpdocTextPreview(row.content ?? '') }
            ),
          default: () => getHelpdocTextTooltip(row.content ?? ''),
        }
      ),
  },
  { title: '分类', key: 'packageName', width: 150, ellipsis: { tooltip: true } },
];

function queryItems() {
  itemQuery.value.pageNum = 1;
  appliedItemQuery.value = { ...itemQuery.value };
  void itemsQuery.refetch();
}

function resetItemQuery() {
  itemQuery.value = createDefaultHelpdocItemQuery();
  appliedItemQuery.value = { ...itemQuery.value };
  void itemsQuery.refetch();
}

async function submitUpload(files: File[]) {
  await enqueueUploads(files, 'helpdoc', {
    group: uploadGroup.value,
    onComplete: async task => {
      message.success(`上传完成：${task.filename}`);
      needReload.value = true;
      await queryClient.invalidateQueries({ queryKey: getSdApiV2HelpdocTreeQueryKey() });
    },
    onError: task => {
      message.error(`上传失败：${task.filename}`);
    },
  });
  uploadDialogVisible.value = false;
  uploadFiles.value = [];
}

function deleteFiles() {
  const keys = checkedFileKeys.value.map(String);
  if (!keys.length) {
    message.error('未选择文件');
    return;
  }
  const summary = summarizeHelpdocDelete(docTree.value, keys);
  dialog.warning({
    title: '删除',
    content: () =>
      h('div', { class: 'delete-summary' }, [
        h('p', `将删除 ${summary.count} 个选中项：`),
        h(
          'ul',
          summary.labels.map(label => h('li', label))
        ),
        h('p', '删除的内容无法找回。'),
      ]),
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(keys);
    },
  });
}

function addAlias(groupKey: string, alias: string) {
  const value = alias.trim();
  if (!value) return;
  if (Array.from(currentAliases.value.values()).some(aliases => aliases.includes(value))) {
    message.error(`别名 ${value} 已被使用`);
    return;
  }
  const next = cloneHelpdocAliases(currentAliases.value);
  next.set(groupKey, [...(next.get(groupKey) ?? []), value]);
  currentAliases.value = next;
}

function removeAlias(groupKey: string, alias: string) {
  const next = cloneHelpdocAliases(currentAliases.value);
  next.set(
    groupKey,
    (next.get(groupKey) ?? []).filter(item => item !== alias)
  );
  currentAliases.value = next;
}

async function saveConfig() {
  await saveConfigMutation.mutateAsync();
}
</script>

<style scoped>
.helpdoc-tabs {
  padding-bottom: 2rem;
}

:deep(.help-content-preview) {
  display: block;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

@media screen and (max-width: 639.9px) {
  .helpdoc-tabs :deep(.n-tabs-nav-scroll-content) {
    justify-content: flex-start !important;
  }
}
</style>
