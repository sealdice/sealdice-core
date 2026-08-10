import { computed, nextTick, ref, toValue, watch, type MaybeRefOrGetter } from 'vue';
import { cloneDeep } from 'es-toolkit/compat';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useDialog, useMessage } from 'naive-ui';
import {
  getSdApiV2CustomTextOptions,
  getSdApiV2CustomTextQueryKey,
  postSdApiV2CustomTextByCategoryPreviewRefresh,
  putSdApiV2CustomTextByCategory,
  type TextItemCompatibleInfo,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';
import { copyText } from '@/features/clipboard';
import { getErrorMessage } from '@/features/auth/error';
import { useUnsavedChanges } from '@/features/unsavedChanges';
import {
  normalizeCustomTextData,
  normalizeTextDict,
  type TextTemplateItem,
  type TextTemplateWithWeightDict,
} from './types';
import {
  buildCustomTextExportContent,
  createTextItemKeyStore,
  getCustomTextGroups,
  parseCustomTextImportContent,
  sortCustomTextCategory,
  type CustomTextFilterMode,
} from './viewModel';

export function useCustomTextEditor(categorySource: MaybeRefOrGetter<string>) {
  const message = useMessage();
  const dialog = useDialog();
  const queryClient = useQueryClient();

  const category = computed(() => toValue(categorySource));
  const texts = ref<TextTemplateWithWeightDict>({});
  const configForImport = ref('');
  const importOnlyCurrent = ref(true);
  const importImpact = ref(true);
  const dialogImportVisible = ref(false);
  const initialTexts = ref<TextTemplateWithWeightDict>({});
  const filterMode = ref<CustomTextFilterMode>('all');
  const currentFilterGroup = ref('');
  const currentFilterName = ref('');
  const textItemKeys = createTextItemKeyStore();

  const customTextQuery = useQuery({
    ...getSdApiV2CustomTextOptions(),
    enabled: hasAccessToken,
  });

  const remoteData = computed(() => normalizeCustomTextData(customTextQuery.data.value?.item));
  const helpInfo = computed(() => remoteData.value.helpInfo);
  const previewInfo = computed(() => remoteData.value.previewInfo);
  const hasCategory = computed(() => Boolean(texts.value[category.value]));
  const modified = computed(() => {
    if (!category.value || !texts.value[category.value]) return false;
    return JSON.stringify(texts.value[category.value] ?? {}) !== JSON.stringify(initialTexts.value[category.value] ?? {});
  });
  const filterGroups = computed(() => getCustomTextGroups(helpInfo.value[category.value]));
  const sortedCategory = computed(() =>
    sortCustomTextCategory({
      texts: texts.value,
      helpInfo: helpInfo.value,
      category: category.value,
      filterMode: filterMode.value,
      filterName: currentFilterName.value,
      filterGroup: currentFilterGroup.value,
    }),
  );

  const saveMutation = useMutation({
    mutationFn: async (targetCategory: string) => {
      const { data } = await putSdApiV2CustomTextByCategory({
        path: { category: targetCategory },
        body: { data: texts.value[targetCategory] ?? {} },
        throwOnError: true,
      });
      return data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomTextQueryKey() });
    },
  });

  const previewRefreshMutation = useMutation({
    mutationFn: async (targetCategory: string) => {
      const { data } = await postSdApiV2CustomTextByCategoryPreviewRefresh({
        path: { category: targetCategory },
        throwOnError: true,
      });
      return data;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomTextQueryKey() });
    },
  });

  function syncLocalTexts(force = false) {
    const data = customTextQuery.data.value?.item;
    if (!data || (modified.value && !force)) return;
    const nextTexts = cloneDeep(normalizeCustomTextData(data).texts);
    texts.value = nextTexts;
    initialTexts.value = cloneDeep(nextTexts);
  }

  watch(
    () => customTextQuery.data.value?.item,
    () => syncLocalTexts(),
    { immediate: true },
  );

  watch(
    category,
    () => {
      filterMode.value = 'all';
      currentFilterGroup.value = '';
      currentFilterName.value = '';
    },
  );

  watch(
    () => dialogImportVisible.value,
    newValue => {
      if (newValue) {
        importRefresh();
      }
    },
  );

  watch(
    () => [importImpact.value, importOnlyCurrent.value, category.value],
    () => {
      if (dialogImportVisible.value) {
        importRefresh();
      }
    },
  );

  async function copied() {
    try {
      await copyText(configForImport.value);
      message.success('进行了复制！');
    } catch {
      message.error('复制失败');
    }
  }

  function importRefresh() {
    configForImport.value = buildCustomTextExportContent({
      texts: texts.value,
      category: category.value,
      onlyCurrent: importOnlyCurrent.value,
      compact: importImpact.value,
    });
  }

  async function doImport() {
    let normalized: TextTemplateWithWeightDict;
    try {
      normalized = parseCustomTextImportContent(configForImport.value);
    } catch {
      message.error('导入内容格式不正确，请检查 JSON 结构后重试');
      return;
    }

    const savedCategories: string[] = [];
    const failedCategories: string[] = [];
    for (const [targetCategory, value] of Object.entries(normalized)) {
      if (!texts.value[targetCategory]) continue;
      texts.value[targetCategory] = value;
      try {
        await saveMutation.mutateAsync(targetCategory);
        initialTexts.value[targetCategory] = cloneDeep(value);
        savedCategories.push(targetCategory);
      } catch (error) {
        failedCategories.push(`${targetCategory}（${getErrorMessage(error, '保存失败')}）`);
      }
    }

    if (!savedCategories.length && !failedCategories.length) {
      message.warning('导入内容中没有当前可用的文案分类');
      return;
    }

    if (failedCategories.length) {
      const savedText = savedCategories.length ? `已保存：${savedCategories.join('、')}；` : '';
      message.error(`${savedText}保存失败：${failedCategories.join('、')}。请修正后重试`);
      return;
    }

    if (savedCategories.length) {
      message.success(`已保存：${savedCategories.join('、')}`);
      dialogImportVisible.value = false;
    }
  }

  function addItem(keyName: string) {
    texts.value[category.value][keyName].push(['', 1]);
  }

  function doChanged(targetCategory: string, keyName: string) {
    const itemHelpInfo = helpInfo.value[targetCategory]?.[keyName];
    if (itemHelpInfo) {
      itemHelpInfo.modified = true;
    }
  }

  function removeItem(items: TextTemplateItem[], index: number) {
    items.splice(index, 1);
  }

  async function save() {
    await saveMutation.mutateAsync(category.value);
    initialTexts.value[category.value] = cloneDeep(texts.value[category.value] ?? {});
    syncLocalTexts(true);
    message.success('已保存');
  }

  useUnsavedChanges('custom-text', {
    label: computed(() => category.value ? `自定义文案 / ${category.value}` : '自定义文案'),
    dirty: modified,
    save,
    saving: computed(() => saveMutation.isPending.value),
    confirmMessage: computed(() => {
      const target = category.value ? `自定义文案 / ${category.value}` : '自定义文案';
      return `${target} 还有修改，确定要忽略？`;
    }),
  });

  async function refreshPreview() {
    await previewRefreshMutation.mutateAsync(category.value);
    message.success('预览已刷新');
  }

  function getPreview(keyName: string, text: string): TextItemCompatibleInfo | undefined {
    return previewInfo.value[`${category.value}:${keyName}`]?.[text];
  }

  function getPreviewCheckErr(keyName: string, text: string) {
    const info = getPreview(keyName, text);
    if (info) {
      if (info.version === 'v2') return Boolean(info.errV2);
      if (info.version === 'v1') return Boolean(info.errV1);
    }
    return false;
  }

  function deleteValue(targetCategory: string, keyName: string) {
    delete texts.value[targetCategory][keyName];
  }

  function askDeleteValue(targetCategory: string, keyName: string) {
    dialog.warning({
      title: '警告',
      content: '删除这条文本，确定吗？',
      positiveText: '确定',
      negativeText: '取消',
      closable: false,
      onPositiveClick: async () => {
        deleteValue(targetCategory, keyName);
        message.success('成功');
      },
    });
  }

  function resetValue(targetCategory: string, keyName: string) {
    const itemHelpInfo = helpInfo.value[targetCategory]?.[keyName];
    texts.value[targetCategory][keyName] = normalizeTextDict({
      [targetCategory]: {
        [keyName]: itemHelpInfo?.origin ?? [],
      },
    })[targetCategory][keyName];
    if (itemHelpInfo) {
      itemHelpInfo.modified = false;
    }
  }

  function askResetValue(targetCategory: string, keyName: string) {
    dialog.warning({
      title: '警告',
      content: '重置这条文本回默认状态，确定吗？',
      positiveText: '确定',
      negativeText: '取消',
      closable: false,
      onPositiveClick: async () => {
        resetValue(targetCategory, keyName);
        message.success('成功');
      },
    });
  }

  function handleFilterModeChange(newMode: CustomTextFilterMode) {
    if (newMode === 'group') {
      nextTick(() => {
        currentFilterGroup.value = filterGroups.value[0] ?? '';
        currentFilterName.value = '';
      });
    } else {
      currentFilterGroup.value = '';
      currentFilterName.value = '';
    }
  }

  function clearFilters() {
    filterMode.value = 'all';
    currentFilterGroup.value = '';
    currentFilterName.value = '';
  }

  return {
    category,
    texts,
    configForImport,
    importOnlyCurrent,
    importImpact,
    dialogImportVisible,
    filterMode,
    filterGroups,
    currentFilterGroup,
    currentFilterName,
    helpInfo,
    customTextQuery,
    hasCategory,
    sortedCategory,
    saveMutation,
    previewRefreshMutation,
    textItemKeyOf: textItemKeys.keyOf,
    copied,
    importRefresh,
    doImport,
    addItem,
    doChanged,
    removeItem,
    refreshPreview,
    getPreview,
    getPreviewCheckErr,
    askDeleteValue,
    askResetValue,
    handleFilterModeChange,
    clearFilters,
  };
}
