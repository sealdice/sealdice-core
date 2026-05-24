import { computed, nextTick, reactive, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import { useDialog, useMessage, type UploadCustomRequestOptions } from 'naive-ui';
import {
  deleteSdApiV2CustomReplyFilesByFilename,
  getSdApiV2ConfigReplyOptions,
  getSdApiV2ConfigReplyQueryKey,
  getSdApiV2CustomReplyFiles,
  getSdApiV2CustomReplyFilesByFilename,
  getSdApiV2CustomReplyFilesByFilenameConditions,
  getSdApiV2CustomReplyFilesByFilenameDownload,
  getSdApiV2CustomReplyFilesByFilenameRules,
  postSdApiV2CustomReplyFiles,
  postSdApiV2CustomReplyFilesUpload,
  putSdApiV2ConfigReply,
  putSdApiV2CustomReplyFilesByFilename,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import { hasAccessToken } from '@/features/auth/state';
import { useUnsavedChanges } from '@/features/unsavedChanges';
import { parseReplyImportText } from './importParser';
import {
  cloneReplyFileDraft,
  cloneReplyTask,
  normalizeCondition,
  normalizeReplyFileDetail,
  normalizeReplyTask,
  toApiReplyConfig,
  type ReplyCondition,
  type ReplyFileDraft,
  type ReplyTask,
} from './model';

export type ReplyFileQuery = {
  page: number;
  pageSize: number;
  keyword: string;
  sortBy: string;
  sortOrder: string;
};

export function useCustomReplyEditor() {
  const message = useMessage();
  const dialog = useDialog();
  const queryClient = useQueryClient();

  const selectedFilename = ref('');
  const commonConditionsPage = ref(1);
  const commonConditionsPageSize = ref(10);
  const rulesPage = ref(1);
  const rulesPageSize = ref(20);
  const rulePageItems = ref<ReplyTask[]>([]);
  const modified = ref(false);
  const syncingRemote = ref(false);
  const importDialogVisible = ref(false);
  const licenseDialogVisible = ref(false);
  const pendingReplyEnable = ref<boolean | null>(null);
  const newFileDialogVisible = ref(false);
  const newFilename = ref(`reply${Math.ceil(Math.random() * 10000)}.yaml`);
  const configForImport = ref('');

  const fileQuery = reactive<ReplyFileQuery>({
    page: 1,
    pageSize: 30,
    keyword: '',
    sortBy: 'updateTime',
    sortOrder: 'desc',
  });

  const drafts = ref<Record<string, ReplyFileDraft>>({});
  const initialDrafts = ref<Record<string, ReplyFileDraft>>({});
  const customReplyFileListQueryKey = computed(() => [
    'custom-reply-files',
    fileQuery.page,
    fileQuery.pageSize,
    fileQuery.keyword,
    fileQuery.sortBy,
    fileQuery.sortOrder,
  ]);

  const replyConfigQuery = useQuery({
    ...getSdApiV2ConfigReplyOptions(),
    enabled: hasAccessToken,
  });

  const fileListQuery = useQuery({
    queryKey: customReplyFileListQueryKey,
    enabled: hasAccessToken,
    queryFn: async () => {
      const { data } = await getSdApiV2CustomReplyFiles({
        query: {
          page: fileQuery.page,
          pageSize: fileQuery.pageSize,
          keyword: fileQuery.keyword,
          sortBy: fileQuery.sortBy,
          sortOrder: fileQuery.sortOrder,
        },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const currentFileDetailQuery = useQuery({
    queryKey: ['custom-reply-file-detail', selectedFilename],
    enabled: computed(() => hasAccessToken.value && selectedFilename.value !== ''),
    queryFn: async () => {
      const { data } = await getSdApiV2CustomReplyFilesByFilename({
        path: { filename: selectedFilename.value },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const currentConditionsPageQuery = useQuery({
    queryKey: ['custom-reply-file-conditions', selectedFilename, commonConditionsPage, commonConditionsPageSize],
    enabled: computed(() => hasAccessToken.value && selectedFilename.value !== ''),
    queryFn: async () => {
      const { data } = await getSdApiV2CustomReplyFilesByFilenameConditions({
        path: { filename: selectedFilename.value },
        query: {
          page: commonConditionsPage.value,
          pageSize: commonConditionsPageSize.value,
        },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const currentRulesPageQuery = useQuery({
    queryKey: ['custom-reply-file-rules', selectedFilename, rulesPage, rulesPageSize],
    enabled: computed(() => hasAccessToken.value && selectedFilename.value !== ''),
    queryFn: async () => {
      const { data } = await getSdApiV2CustomReplyFilesByFilenameRules({
        path: { filename: selectedFilename.value },
        query: {
          page: rulesPage.value,
          pageSize: rulesPageSize.value,
        },
        throwOnError: true,
      });
      return data.item;
    },
  });

  const pageBusy = computed(() => {
    return (
      replyConfigQuery.isFetching.value ||
      fileListQuery.isFetching.value ||
      currentFileDetailQuery.isFetching.value ||
      currentConditionsPageQuery.isFetching.value ||
      currentRulesPageQuery.isFetching.value
    );
  });

  const replyEnabled = computed(() => replyConfigQuery.data.value?.item.customReplyConfigEnable === true);
  const fileItems = computed(() => fileListQuery.data.value?.list ?? []);
  const fileTotal = computed(() => fileListQuery.data.value?.total ?? 0);
  const currentFileDraft = computed(() => {
    if (!selectedFilename.value) return null;
    return drafts.value[selectedFilename.value] ?? null;
  });
  const currentInitialDraft = computed(() => {
    if (!selectedFilename.value) return null;
    return initialDrafts.value[selectedFilename.value] ?? null;
  });
  const rulesTotal = computed(() => currentFileDraft.value?.itemCount ?? currentFileDraft.value?.items.length ?? 0);
  const rulePageStart = computed(() => (rulesPage.value - 1) * rulesPageSize.value);
  const commonConditionsTotal = computed(() => currentFileDraft.value?.conditions.length ?? 0);
  const commonConditionsPageStart = computed(() => (commonConditionsPage.value - 1) * commonConditionsPageSize.value);
  const pagedCommonConditions = computed<ReplyCondition[]>({
    get: () => {
      if (!currentFileDraft.value) return [];
      return currentFileDraft.value.conditions.slice(
        commonConditionsPageStart.value,
        commonConditionsPageStart.value + commonConditionsPageSize.value,
      );
    },
    set: value => {
      if (!currentFileDraft.value || syncingRemote.value) return;
      currentFileDraft.value.conditions.splice(commonConditionsPageStart.value, value.length, ...value);
      markModified();
    },
  });

  const replyConfigMutation = useMutation({
    mutationFn: async (enabled: boolean) => {
      const { data } = await putSdApiV2ConfigReply({
        body: { customReplyConfigEnable: enabled },
        throwOnError: true,
      });
      return data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: getSdApiV2ConfigReplyQueryKey() });
    },
  });

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!currentFileDraft.value) throw new Error('missing current file draft');
      const { data } = await putSdApiV2CustomReplyFilesByFilename({
        path: { filename: selectedFilename.value },
        body: toApiReplyConfig(currentFileDraft.value),
        throwOnError: true,
      });
      return data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-file-detail'] });
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-file-conditions'] });
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-file-rules'] });
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-files'] });
      if (currentFileDraft.value && selectedFilename.value) {
        initialDrafts.value = {
          ...initialDrafts.value,
          [selectedFilename.value]: cloneReplyFileDraft(currentFileDraft.value),
        };
      }
      message.success('已保存');
    },
    onError: () => {
      message.error('保存失败');
    },
  });

  watch(
    [currentFileDraft, currentInitialDraft],
    () => {
      const current = currentFileDraft.value;
      const initial = currentInitialDraft.value;
      if (!current || !initial) {
        modified.value = false;
        return;
      }
      modified.value = JSON.stringify(current) !== JSON.stringify(initial);
    },
    { deep: true, immediate: true },
  );

  useUnsavedChanges('custom-reply', {
    label: computed(() => {
      const filename = selectedFilename.value || currentFileDraft.value?.filename;
      return filename ? `自定义回复 / ${filename}` : '自定义回复';
    }),
    dirty: computed(() => modified.value),
    save: saveCurrent,
    saving: computed(() => saveMutation.isPending.value),
    canSave: computed(() => Boolean(currentFileDraft.value) && modified.value),
    confirmMessage: computed(() => {
      const filename = selectedFilename.value || currentFileDraft.value?.filename;
      return filename
        ? `自定义回复 / ${filename} 还有修改，确定要忽略？`
        : '自定义回复还有修改，确定要忽略？';
    }),
  });

  watch(
    fileItems,
    items => {
      if (!items.length) {
        selectedFilename.value = '';
        return;
      }
      if (!items.some(item => item.filename === selectedFilename.value)) {
        selectedFilename.value = items[0]?.filename ?? '';
      }
    },
    { immediate: true },
  );

  watch(
    () => currentFileDetailQuery.data.value,
    detail => {
      if (!detail) return;
      syncingRemote.value = true;
      const normalized = normalizeReplyFileDetail(detail);
      const existing = drafts.value[detail.filename];
      drafts.value = {
        ...drafts.value,
        [detail.filename]: existing ?? normalized,
      };
      initialDrafts.value = {
        ...initialDrafts.value,
        [detail.filename]: cloneReplyFileDraft(normalized),
      };
      modified.value = false;
      void nextTick(() => {
        syncingRemote.value = false;
      });
    },
    { immediate: true },
  );

  watch(
    () => currentConditionsPageQuery.data.value,
    pageData => {
      if (!pageData || !currentFileDraft.value) return;
      syncingRemote.value = true;
      const draft = currentFileDraft.value;
      ensureConditionLength(draft, pageData.total);
      for (const condition of pageData.list ?? []) {
        draft.conditions[condition.index] = normalizeCondition(condition.item);
      }
      void nextTick(() => {
        syncingRemote.value = false;
      });
    },
    { immediate: true },
  );

  watch(
    () => currentRulesPageQuery.data.value,
    pageData => {
      if (!pageData || !currentFileDraft.value) return;
      syncingRemote.value = true;
      const draft = currentFileDraft.value;
      ensureDraftItemLength(draft, pageData.total);
      for (const rule of pageData.list ?? []) {
        draft.items[rule.index] = normalizeReplyTask(rule.item);
      }
      draft.itemCount = pageData.total;
      syncRulePageItemsFromDraft();
      void nextTick(() => {
        syncingRemote.value = false;
      });
    },
    { immediate: true },
  );

  watch(selectedFilename, () => {
    commonConditionsPage.value = 1;
    rulesPage.value = 1;
    rulePageItems.value = [];
  });

  watch(
    [rulesPage, rulesPageSize, currentFileDraft],
    () => {
      syncRulePageItemsFromDraft();
    },
  );

  watch(
    rulePageItems,
    value => {
      if (!currentFileDraft.value || syncingRemote.value) return;
      const start = rulePageStart.value;
      value.forEach((item, index) => {
        currentFileDraft.value!.items[start + index] = cloneReplyTask(item);
      });
      currentFileDraft.value.itemCount = currentFileDraft.value.items.filter(Boolean).length;
      modified.value = true;
    },
    { deep: true },
  );

  function updateFileQuery(nextQuery: ReplyFileQuery) {
    Object.assign(fileQuery, nextQuery);
  }

  function selectFile(filename: string) {
    if (selectedFilename.value === filename) return;
    selectedFilename.value = filename;
  }

  function handleReplySwitchUpdate(value: boolean) {
    if (value === replyEnabled.value) return;
    if (value) {
      pendingReplyEnable.value = true;
      licenseDialogVisible.value = true;
      return;
    }
    pendingReplyEnable.value = null;
    void toggleReplyEnabled(false);
  }

  async function toggleReplyEnabled(value: boolean) {
    try {
      await replyConfigMutation.mutateAsync(value);
    } catch {
      message.error('总开关更新失败');
    }
  }

  async function acceptLicense() {
    if (!pendingReplyEnable.value) {
      licenseDialogVisible.value = false;
      return;
    }
    await toggleReplyEnabled(true);
    licenseDialogVisible.value = false;
    pendingReplyEnable.value = null;
  }

  function refuseLicense() {
    licenseDialogVisible.value = false;
    pendingReplyEnable.value = null;
  }

  function addCommonCondition() {
    currentFileDraft.value?.conditions.push({
      condType: 'textMatch',
      matchType: 'matchExact',
      value: '要匹配的文本',
    });
    const lastPage = Math.max(1, Math.ceil(commonConditionsTotal.value / commonConditionsPageSize.value));
    commonConditionsPage.value = lastPage;
    markModified();
  }

  function deleteCommonCondition(index: number) {
    if (!currentFileDraft.value) return;
    const absoluteIndex = commonConditionsPageStart.value + index;
    currentFileDraft.value.conditions.splice(absoluteIndex, 1);
    const lastPage = Math.max(1, Math.ceil(commonConditionsTotal.value / commonConditionsPageSize.value));
    if (commonConditionsPage.value > lastPage) {
      commonConditionsPage.value = lastPage;
    }
    markModified();
  }

  function toggleCurrentFileEnable() {
    if (!currentFileDraft.value) return;
    currentFileDraft.value.enable = !currentFileDraft.value.enable;
    markModified();
  }

  function getFileEnableStatus(filename: string, fallback: boolean) {
    return drafts.value[filename]?.enable ?? fallback;
  }

  function addReplyItem() {
    currentFileDraft.value?.items.push({
      enable: true,
      conditions: [{ condType: 'textMatch', matchType: 'matchExact', value: '要匹配的文本' }],
      results: [{ resultType: 'replyToSender', delay: 0, message: [['说点什么', 1]] }],
    });
    if (currentFileDraft.value) {
      currentFileDraft.value.itemCount = currentFileDraft.value.items.filter(Boolean).length;
    }
    const lastPage = Math.max(1, Math.ceil(rulesTotal.value / rulesPageSize.value));
    rulesPage.value = lastPage;
    syncRulePageItemsFromDraft();
    markModified();
  }

  function deleteReplyItem(index: number) {
    if (!currentFileDraft.value) return;
    const absoluteIndex = rulePageStart.value + index;
    currentFileDraft.value.items.splice(absoluteIndex, 1);
    currentFileDraft.value.itemCount = currentFileDraft.value.items.filter(Boolean).length;
    const lastPage = Math.max(1, Math.ceil(rulesTotal.value / rulesPageSize.value));
    if (rulesPage.value > lastPage) {
      rulesPage.value = lastPage;
    }
    syncRulePageItemsFromDraft();
    markModified();
  }

  async function saveCurrent() {
    await saveMutation.mutateAsync();
  }

  async function createNewFile() {
    try {
      const target = newFilename.value.trim() || `reply${Math.ceil(Math.random() * 10000)}.yaml`;
      const { data } = await postSdApiV2CustomReplyFiles({
        body: { filename: target },
        throwOnError: true,
      });
      if (!data.item.success) {
        message.error('创建失败，可能已存在同名文件');
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-files'] });
      newFileDialogVisible.value = false;
      newFilename.value = `reply${Math.ceil(Math.random() * 10000)}.yaml`;
      selectFile(target);
      message.success('创建成功');
    } catch {
      message.error('创建失败');
    }
  }

  function deleteCurrentFile() {
    if (!selectedFilename.value) return;
    dialog.warning({
      title: '删除文件',
      content: '是否删除此文件？',
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await deleteSdApiV2CustomReplyFilesByFilename({
            path: { filename: selectedFilename.value },
            throwOnError: true,
          });
          const filename = selectedFilename.value;
          const nextDrafts = { ...drafts.value };
          delete nextDrafts[filename];
          drafts.value = nextDrafts;
          selectedFilename.value = '';
          await queryClient.invalidateQueries({ queryKey: ['custom-reply-files'] });
          message.success('删除成功');
        } catch {
          message.error('删除失败');
        }
      },
    });
  }

  async function downloadCurrentFile() {
    if (!selectedFilename.value) return;
    await downloadApiFile(
      getSdApiV2CustomReplyFilesByFilenameDownload({
        path: { filename: selectedFilename.value },
        responseType: 'blob',
        throwOnError: true,
      }),
      selectedFilename.value,
    );
  }

  async function uploadFile(options: UploadCustomRequestOptions) {
    try {
      await postSdApiV2CustomReplyFilesUpload({
        body: { file: options.file.file as File },
        throwOnError: true,
      });
      await queryClient.invalidateQueries({ queryKey: ['custom-reply-files'] });
      message.success('上传完成');
      options.onFinish();
    } catch {
      message.error('上传失败，可能有同名文件');
      options.onError();
    }
  }

  function formatUpdateTime(ts: number): string {
    if (!ts) return '未记录';
    return dayjs.unix(ts).format('MM-DD HH:mm');
  }

  function doImport() {
    if (!currentFileDraft.value) return;
    const importedTasks = parseReplyImportText(configForImport.value);
    currentFileDraft.value.items.push(...importedTasks);
    configForImport.value = '';
    importDialogVisible.value = false;
    currentFileDraft.value.itemCount = currentFileDraft.value.items.filter(Boolean).length;
    const lastPage = Math.max(1, Math.ceil(rulesTotal.value / rulesPageSize.value));
    rulesPage.value = lastPage;
    syncRulePageItemsFromDraft();
    markModified();
    message.success('导入成功！');
  }

  function markModified() {
    if (syncingRemote.value) return;
    const current = currentFileDraft.value;
    const initial = currentInitialDraft.value;
    modified.value = !!current && !!initial && JSON.stringify(current) !== JSON.stringify(initial);
  }

  function ensureDraftItemLength(draft: ReplyFileDraft, total: number) {
    if (draft.items.length >= total) return;
    draft.items.length = total;
  }

  function ensureConditionLength(draft: ReplyFileDraft, total: number) {
    if (draft.conditions.length >= total) return;
    draft.conditions.length = total;
  }

  function syncRulePageItemsFromDraft() {
    if (!currentFileDraft.value) {
      rulePageItems.value = [];
      return;
    }
    syncingRemote.value = true;
    rulePageItems.value = currentFileDraft.value.items
      .slice(rulePageStart.value, rulePageStart.value + rulesPageSize.value)
      .filter(Boolean)
      .map(item => cloneReplyTask(item));
    void nextTick(() => {
      syncingRemote.value = false;
    });
  }

  return {
    selectedFilename,
    commonConditionsPage,
    commonConditionsPageSize,
    rulesPage,
    rulesPageSize,
    rulePageItems,
    modified,
    importDialogVisible,
    licenseDialogVisible,
    newFileDialogVisible,
    newFilename,
    configForImport,
    fileQuery,
    replyConfigMutation,
    saveMutation,
    pageBusy,
    replyEnabled,
    fileItems,
    fileTotal,
    currentFileDraft,
    rulesTotal,
    rulePageStart,
    commonConditionsTotal,
    pagedCommonConditions,
    updateFileQuery,
    selectFile,
    handleReplySwitchUpdate,
    acceptLicense,
    refuseLicense,
    addCommonCondition,
    deleteCommonCondition,
    toggleCurrentFileEnable,
    getFileEnableStatus,
    addReplyItem,
    deleteReplyItem,
    saveCurrent,
    createNewFile,
    deleteCurrentFile,
    downloadCurrentFile,
    uploadFile,
    formatUpdateTime,
    doImport,
    markModified,
  };
}
