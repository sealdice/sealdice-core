<template>
  <main class="custom-text-page">
    <n-spin :show="customTextQuery.isFetching.value && !customTextQuery.data.value">
      <TipBox type="info">
        <n-collapse>
          <n-collapse-item name="1">
            <template #header>
              <n-text tag="strong">查看帮助</n-text>
            </template>

            <n-text tag="p">
              <div>此处可以对骰子返回的文本进行修改。最终返回的文本将为多个条目中随机抽取的一个。</div>
              <div>
                随机文本：默认一种显示结果，如果需要多种反馈结果，使用＋添加条目，使用 - 删除条目
              </div>
              <div>
                遇到有此标记 (<n-icon><i-carbon-paint-brush /></n-icon>)
                的条目，说明和默认值不同，是一个自定义条目
              </div>
              <div style="margin-top: 1rem">
                文本下方的
                <n-tag size="small" :bordered="false">标签</n-tag>
                代表了被默认文本所使用的特殊变量，你可以使用 {变量名} 来插入他们，例如 {$t判定值}
              </div>
              <div>
                除此之外，这些变量可以在所有文本中使用：
                <n-flex size="small" wrap>
                  <n-tag
                    :key="i"
                    size="small"
                    :bordered="false"
                    v-for="i in [
                      '$t玩家',
                      '$tQQ昵称',
                      '$t个人骰子面数',
                      '$tQQ',
                      '$t骰子帐号',
                      '$t骰子昵称',
                      '$t群号',
                      '$t群名',
                    ]"
                  >
                    {{ i }}
                  </n-tag>
                </n-flex>
              </div>
              <div>
                <span>以及，所有的自定义文本都可以嵌套使用，例如：</span>
                <div>
                  <b>这里是{核心：骰子名字}，我是一个示例</b>
                </div>
                <div>默认会被解析为：</div>
                <div>
                  <b>这里是海豹 bot，我是一个示例</b>
                </div>
                <div>注意！千万不要递归嵌套，会发生很糟糕的事情。</div>
              </div>

              <div style="margin-top: 1rem">
                <div>此外，支持插入图片，将图片放在骰子的适当目录，再写这样一句话即可：</div>
                <div><n-text code>[图:data/images/sealdice.png]</n-text></div>
                <div>可以参考 核心：骰子进群 词条</div>
                <div>同样的，可以使用 CQ 码插入图片和其他内容，关于 CQ 码，请参阅 onebot 项目文档</div>
              </div>

              <div style="margin-top: 1rem">
                <b>
                  COC 的“判定 - 常规”和“判定 - 简短”主要区别是，多重检定会默认使用简短版本 (.ra
                  3#射击)
                </b>
                <b>进行调整后，可以在左侧面板“指令测试”中进行测试！</b>
              </div>
            </n-text>
          </n-collapse-item>
        </n-collapse>
      </TipBox>

      <div class="custom-text-toolbar">
        <n-flex align="center" size="small" class="custom-text-search">
          <n-text>搜索：</n-text>
          <span class="custom-text-search-input">
            <n-input size="small" v-model:value="currentFilterName" clearable>
              <template #prefix>
                <n-icon><i-carbon-search /></n-icon>
              </template>
            </n-input>
          </span>
        </n-flex>
        <n-flex align="center" size="small" class="custom-text-actions">
          <n-button
            type="info"
            secondary
            :loading="previewRefreshMutation.isPending.value"
            @click="refreshPreview"
          >
            刷新预览
          </n-button>
          <n-button type="info" secondary @click="dialogImportVisible = true">导入/导出</n-button>
        </n-flex>
      </div>

      <n-flex class="custom-text-filter-row mb-8 mt-4" align="center" wrap>
        <n-radio-group v-model:value="filterMode" @update:value="handleFilterModeChange">
          <n-radio
            v-for="mode of filterModes"
            :key="mode.value"
            :value="mode.value"
            :label="mode.desc"
          />
        </n-radio-group>
        <n-flex v-if="filterMode === 'group'" align="center" class="custom-text-group-filter">
          <n-text>分组：</n-text>
          <span class="custom-text-group-select">
            <n-select
              v-model:value="currentFilterGroup"
              filterable
              tag
              :options="
                filterGroups.map(group => {
                  return { label: group, value: group };
                })
              "
            />
          </span>
        </n-flex>
      </n-flex>

      <n-empty v-if="!hasCategory" description="未找到当前文案分类" />

      <n-collapse v-else class="text-collapse" :default-expanded-names="['__others__']">
        <CustomTextBox
          :key="group"
          v-for="[group, values] in sortedCategory"
          :group="group"
        >
          <template #values>
            <n-grid x-gap="24" y-gap="16" cols="1 m:2" responsive="screen">
              <n-grid-item v-for="[keyName, items] in values" :key="keyName">
                <n-form ref="form" label-width="auto" label-position="top">
                  <n-form-item class="w-full">
                    <template #label>
                      <div>
                        <n-tag
                          type="default"
                          size="small"
                          style="margin-right: 0.5rem"
                          :bordered="false"
                        >
                          {{
                            helpInfo[category]?.[keyName]?.subType ||
                            (helpInfo[category]?.[keyName]?.notBuiltin ? '旧版文本' : '其它')
                          }}
                        </n-tag>

                        <span>
                          <span>{{ keyName }}</span>
                          <n-tooltip v-if="helpInfo[category]?.[keyName]?.extraText">
                            <template #trigger>
                              <n-icon><i-carbon-help-filled /></n-icon>
                            </template>
                            {{ helpInfo[category]?.[keyName]?.extraText }}
                          </n-tooltip>
                        </span>

                        <template v-if="helpInfo[category]?.[keyName]?.notBuiltin">
                          <n-tooltip placement="bottom-end">
                            <template #trigger>
                              <n-icon
                                style="float: right; margin-left: 1rem"
                                @click="askDeleteValue(category, keyName)"
                              >
                                <i-carbon-row-delete />
                              </n-icon>
                            </template>
                            移除 - 这个文本在新版的默认配置中不被使用，<br />
                            但升级而来时仍可能被使用，请确认无用后删除
                          </n-tooltip>
                        </template>

                        <template v-if="helpInfo[category]?.[keyName]?.modified">
                          <n-tooltip placement="bottom-end">
                            <template #trigger>
                              <n-icon
                                style="float: right; margin-left: 1rem"
                                @click="askResetValue(category, keyName)"
                              >
                                <i-carbon-paint-brush />
                              </n-icon>
                            </template>
                            重置为初始值
                          </n-tooltip>
                        </template>
                      </div>
                    </template>

                    <n-flex vertical class="w-full">
                      <n-text v-if="keyName === '戳一戳'" type="warning" class="mb-1 text-xs">
                        请确认你使用的 QQ
                        连接方式支持该功能，若不支持请于「基本设置」中关闭戳一戳来避免日志中出现相关报错。
                      </n-text>

                      <div
                        v-for="(item, index) in items"
                        :key="index"
                        style="width: 100%; margin-bottom: 0.5rem"
                      >
                        <n-flex align="center">
                          <div>
                            <n-tooltip placement="bottom-start">
                              <template #trigger>
                                <n-icon>
                                  <i-carbon-add-filled v-if="index === 0" @click="addItem(keyName)" />
                                  <i-carbon-close-outline v-else @click="removeItem(items, index)" />
                                </n-icon>
                              </template>
                              {{
                                index === 0
                                  ? '点击添加一个回复语，SealDice 将会随机抽取一个回复'
                                  : '点击删除你不想要的回复语'
                              }}
                            </n-tooltip>
                          </div>
                          <div class="relative flex-auto">
                            <n-input
                              class="w-full"
                              type="textarea"
                              :autosize="{ minRows: 3 }"
                              v-model:value="item[0]"
                              @update:value="doChanged(category, keyName)"
                            />

                            <div class="absolute bottom-0 right-1" v-if="getPreview(keyName, item[0])">
                              <n-popover placement="bottom-start">
                                <template #trigger>
                                  <span
                                    v-if="getPreviewCheckErr(keyName, item[0])"
                                    class="text-red-500"
                                    style="margin-left: 0.1rem; margin-top: 0.1rem"
                                  >
                                    <n-icon><i-carbon-close-filled /></n-icon>
                                  </span>
                                  <n-flex v-else>
                                    <span
                                      v-if="getPreview(keyName, item[0])?.version === 'v2'"
                                      class="text-blue-500"
                                      style="margin-left: 0.1rem; margin-top: 0.1rem"
                                    >
                                      <n-icon><i-carbon-checkmark-filled /></n-icon>
                                    </span>
                                    <span
                                      v-if="getPreview(keyName, item[0])?.version === 'v1'"
                                      class="text-yellow-500"
                                      style="margin-left: 0.1rem; margin-top: 0.1rem"
                                    >
                                      <n-icon><i-carbon-checkmark-filled /></n-icon>
                                    </span>
                                  </n-flex>
                                </template>

                                <component :is="getPreviewInfo(keyName, item[0])" />
                              </n-popover>
                            </div>
                          </div>
                        </n-flex>
                      </div>
                      <n-flex size="small" wrap>
                        <n-tag
                          size="small"
                          type="info"
                          :bordered="false"
                          v-for="item in helpInfo[category]?.[keyName]?.vars ?? []"
                          :key="item"
                        >
                          {{ item }}
                        </n-tag>
                      </n-flex>
                    </n-flex>
                  </n-form-item>
                </n-form>
              </n-grid-item>
            </n-grid>
          </template>
        </CustomTextBox>
      </n-collapse>

      <n-modal
        v-model:show="dialogImportVisible"
        preset="card"
        title="导入导出"
        :mask-closable="false"
        :close-on-esc="false"
        :closable="false"
        class="the-dialog"
      >
        <template #header-extra>
          <n-flex>
            <n-switch v-model:value="importOnlyCurrent">
              <template #checked>仅当前页面</template>
              <template #unchecked>全部文案</template>
            </n-switch>
            <n-checkbox v-model:checked="importImpact">紧凑</n-checkbox>
          </n-flex>
        </template>
        <n-flex vertical>
          <n-text tag="strong">以下为导出内容，可以复制给别人</n-text>
          <n-input
            placeholder="填入数据"
            type="textarea"
            :autosize="{ minRows: 4 }"
            class="import-edit"
            id="import-edit"
            v-model:value="configForImport"
          />
        </n-flex>

        <template #footer>
          <n-flex>
            <n-button @click="dialogImportVisible = false">返回</n-button>
            <n-button type="warning" @click="configForImport = ''">清空</n-button>
            <n-button type="info" @click="copied">复制</n-button>
            <n-button
              type="primary"
              :loading="saveMutation.isPending.value"
              :disabled="configForImport === ''"
              @click="doImport"
            >
              导入并保存
            </n-button>
          </n-flex>
        </template>
      </n-modal>
    </n-spin>
  </main>
</template>

<script setup lang="tsx">
import { computed, nextTick, ref, watch } from 'vue';
import { cloneDeep, filter, groupBy, map, mapValues, sortBy, startsWith, trim, uniq } from 'es-toolkit/compat';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useDialog, useMessage } from 'naive-ui';
import { useRoute } from 'vue-router';
import {
  getSdApiV2CustomTextOptions,
  getSdApiV2CustomTextQueryKey,
  postSdApiV2CustomTextByCategoryPreviewRefresh,
  putSdApiV2CustomTextByCategory,
  type TextItemCompatibleInfo,
} from '@/api';
import CustomTextBox from '@/components/shared/CustomTextBox.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import {
  normalizeCustomTextData,
  normalizeTextDict,
  type TextTemplateItem,
  type TextTemplateWithWeightDict,
} from '@/features/customText/types';
import { useUnsavedChanges } from '@/features/unsavedChanges';

// 自定义文案页按 category 动态路由进入。
// 页面维护一份可编辑 texts 草稿，并通过 helpInfo/previewInfo 标识默认变量、
// 自定义覆盖和兼容性预览；保存时只提交当前分类，避免误覆盖其它分类。
const props = defineProps<{ category?: string }>();

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const route = useRoute();

const texts = ref<TextTemplateWithWeightDict>({});
const configForImport = ref('');
const importOnlyCurrent = ref(true);
const importImpact = ref(true);
const dialogImportVisible = ref(false);
const initialTexts = ref<TextTemplateWithWeightDict>({});
const filterMode = ref<string>('all');
const filterGroups = ref<string[]>([]);
const currentFilterGroup = ref<string>('');
const currentFilterName = ref<string>('');

const customTextQuery = useQuery({
  ...getSdApiV2CustomTextOptions(),
  enabled: hasAccessToken,
});

const remoteData = computed(() => normalizeCustomTextData(customTextQuery.data.value?.item));
const helpInfo = computed(() => remoteData.value.helpInfo);
const previewInfo = computed(() => remoteData.value.previewInfo);
const category = computed(() => {
  const routeParams = route.params as Record<string, string | string[] | undefined>;
  const routeCategory = routeParams.category;
  const fallback = Array.isArray(routeCategory) ? routeCategory[0] : routeCategory;
  return props.category ?? String(fallback ?? '');
});
const hasCategory = computed(() => Boolean(texts.value[category.value]));
const modified = computed(() => {
  if (!category.value || !texts.value[category.value]) return false;
  return JSON.stringify(texts.value[category.value] ?? {}) !== JSON.stringify(initialTexts.value[category.value] ?? {});
});

const syncLocalTexts = (force = false) => {
  const data = customTextQuery.data.value?.item;
  if (!data || (modified.value && !force)) return;
  const nextTexts = cloneDeep(normalizeCustomTextData(data).texts);
  texts.value = nextTexts;
  initialTexts.value = cloneDeep(nextTexts);
};

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

interface FilterMode {
  value: string;
  desc: string;
}

const filterModes: FilterMode[] = [
  { value: 'all', desc: '全部' },
  { value: 'unmodified', desc: '默认文案' },
  { value: 'modified', desc: '修改过' },
  { value: 'group', desc: '指定分组' },
  { value: 'deprecated', desc: '旧版文本' },
];

const sortedCategory = computed(() => doSort(category.value));

const doSort = (targetCategory: string) => {
  let items = Object.entries(texts.value?.[targetCategory] ?? {});
  const categoryHelpInfo = helpInfo.value[targetCategory] ?? {};

  if (currentFilterName.value !== '') {
    items = items.filter(item => {
      const itemHelp = categoryHelpInfo[item[0]];
      return item[0].includes(currentFilterName.value) || itemHelp?.subType?.includes(currentFilterName.value);
    });
  }

  switch (filterMode.value) {
    case 'all':
      break;
    case 'unmodified':
      items = items.filter(item => !categoryHelpInfo[item[0]]?.modified);
      break;
    case 'modified':
      items = items.filter(item => categoryHelpInfo[item[0]]?.modified);
      break;
    case 'deprecated':
      items = items.filter(item => categoryHelpInfo[item[0]]?.notBuiltin);
      break;
    case 'group':
      filterGroups.value = sortBy(
        uniq(
          Object.values(categoryHelpInfo)
            .map(info => trim(info.subType))
            .filter(subType => subType !== ''),
        ),
      );
      items = items.filter(item => startsWith(trim(categoryHelpInfo[item[0]]?.subType), currentFilterGroup.value));
      break;
  }

  const boxedGroups = map(
    filter(
      Object.entries(
        groupBy(
          map(items, item => {
            const subType = categoryHelpInfo[item[0]]?.subType ?? '';
            return subType.split(' ')[0] || '__others__';
          }),
          group => group,
        ),
      ),
      group => group?.[1]?.length >= 4,
    ),
    group => group[0],
  );

  return Object.entries(
    mapValues(
      groupBy(items, item => {
        const group = (categoryHelpInfo[item[0]]?.subType ?? '').split(' ')[0] || '__others__';
        if (boxedGroups.includes(group)) {
          return group;
        }
        return '__others__';
      }),
      groupedItems =>
        groupedItems.sort((a, b) => {
          const itemA = categoryHelpInfo[a[0]];
          const itemB = categoryHelpInfo[b[0]];

          if ((itemA?.topOrder ?? 0) !== (itemB?.topOrder ?? 0)) {
            return (itemB?.topOrder ?? 0) - (itemA?.topOrder ?? 0);
          }

          if ((itemA?.subType ?? '') !== (itemB?.subType ?? '')) {
            return (itemB?.subType ?? '').localeCompare(itemA?.subType ?? '');
          }

          return 0;
        }),
    ),
  ).sort((a, b) => {
    const [aGroup] = a;
    const [bGroup] = b;
    if (aGroup === '__others__') {
      return -1;
    }
    if (bGroup === '__others__') {
      return 1;
    }
    return 0;
  });
};

const copied = async () => {
  try {
    await navigator.clipboard.writeText(configForImport.value);
    message.success('进行了复制！');
  } catch {
    message.error('复制失败');
  }
};

const importRefresh = () => {
  const indent = !importImpact.value ? 2 : 0;
  if (importOnlyCurrent.value) {
    configForImport.value = JSON.stringify(
      {
        title: '某人的自定义配置',
        items: {
          [category.value]: texts.value[category.value],
        },
      },
      null,
      indent,
    );
  } else {
    configForImport.value = JSON.stringify(texts.value, null, indent);
  }
};

const doImport = async () => {
  try {
    const data = JSON.parse(configForImport.value) as {
      title?: string;
      items?: TextTemplateWithWeightDict;
    };
    if (!(data.title && data.items)) {
      message.error('格式不正确');
      return;
    }

    const normalized = normalizeTextDict(data.items);
    for (const [targetCategory, value] of Object.entries(normalized)) {
      if (!texts.value[targetCategory]) {
        continue;
      }
      texts.value[targetCategory] = value;
      await saveMutation.mutateAsync(targetCategory);
      initialTexts.value[targetCategory] = cloneDeep(value);
    }
    syncLocalTexts(true);
    message.success('已保存');
    dialogImportVisible.value = false;
  } catch {
    message.error('格式不正确');
  }
};

watch(
  () => dialogImportVisible.value,
  newValue => {
    if (newValue) {
      importRefresh();
    }
  },
);

watch(
  () => [importImpact.value, importOnlyCurrent.value],
  () => {
    importRefresh();
  },
);

const addItem = (keyName: string) => {
  texts.value[category.value][keyName].push(['', 1]);
};

const doChanged = (targetCategory: string, keyName: string) => {
  const itemHelpInfo = helpInfo.value[targetCategory]?.[keyName];
  if (itemHelpInfo) {
    itemHelpInfo.modified = true;
  }
};

const removeItem = (items: TextTemplateItem[], index: number) => {
  items.splice(index, 1);
};

const save = async () => {
  await saveMutation.mutateAsync(category.value);
  initialTexts.value[category.value] = cloneDeep(texts.value[category.value] ?? {});
  syncLocalTexts(true);
  message.success('已保存');
};

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

const refreshPreview = async () => {
  await previewRefreshMutation.mutateAsync(category.value);
  message.success('预览已刷新');
};

const getPreview = (keyName: string, text: string): TextItemCompatibleInfo | undefined => {
  return previewInfo.value[`${category.value}:${keyName}`]?.[text];
};

const getPreviewCheckErr = (keyName: string, text: string) => {
  const info = getPreview(keyName, text);
  if (info) {
    if (info.version === 'v2') return Boolean(info.errV2);
    if (info.version === 'v1') return Boolean(info.errV1);
  }
  return false;
};

const getPreviewInfo = (keyName: string, text: string) => {
  const info = getPreview(keyName, text);
  if (info) {
    let version = info.version;

    if (version === 'v1') {
      version = 'v1 [建议修改]';
    }
    const exists = info.presetExists ? '是' : '否';

    return (
      <div>
        <n-descriptions
          label-placement='left'
          label-align='left'
          separator=' '
          column={1}
          content-class='whitespace-nowrap break-words'
        >
          <n-descriptions-item>
            {{
              label: () => (
                <n-tag type='success' size='small' bordered={false}>
                  引擎版本
                </n-tag>
              ),
              default: () => version,
            }}
          </n-descriptions-item>
          <n-descriptions-item>
            {{
              label: () => (
                <n-tag type='info' size='small' bordered={false}>
                  V2 预览
                </n-tag>
              ),
              default: () => info.textV2 || info.errV2,
            }}
          </n-descriptions-item>
          <n-descriptions-item>
            {{
              label: () => (
                <n-tag type='warning' size='small' bordered={false}>
                  V1 预览
                </n-tag>
              ),
              default: () => info.textV1 || info.errV1,
            }}
          </n-descriptions-item>
          <n-descriptions-item>
            {{
              label: () => (
                <n-tag type='success' size='small' bordered={false}>
                  存在预设
                </n-tag>
              ),
              default: () => exists + ' [存在时预览较为可靠]',
            }}
          </n-descriptions-item>
        </n-descriptions>
      </div>
    );
  }
};

const deleteValue = async (targetCategory: string, keyName: string) => {
  delete texts.value[targetCategory][keyName];
};

const askDeleteValue = async (targetCategory: string, keyName: string) => {
  dialog.warning({
    title: '警告',
    content: '删除这条文本，确定吗？',
    positiveText: '确定',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      await deleteValue(targetCategory, keyName);
      message.success('成功');
    },
  });
};

const resetValue = async (targetCategory: string, keyName: string) => {
  const itemHelpInfo = helpInfo.value[targetCategory]?.[keyName];
  texts.value[targetCategory][keyName] = normalizeTextDict({
    [targetCategory]: {
      [keyName]: itemHelpInfo?.origin ?? [],
    },
  })[targetCategory][keyName];
  if (itemHelpInfo) {
    itemHelpInfo.modified = false;
  }
};

const askResetValue = async (targetCategory: string, keyName: string) => {
  dialog.warning({
    title: '警告',
    content: '重置这条文本回默认状态，确定吗？',
    positiveText: '确定',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      await resetValue(targetCategory, keyName);
      message.success('成功');
    },
  });
};

const handleFilterModeChange = (newMode: string) => {
  if (newMode === 'group') {
    nextTick(() => {
      currentFilterGroup.value = filterGroups.value[0] ?? '';
      currentFilterName.value = '';
    });
  } else {
    currentFilterGroup.value = '';
    currentFilterName.value = '';
  }
};
</script>

<style scoped>
.custom-text-page {
  max-width: 1180px;
  margin: 0 auto;
  text-align: left;
}

.custom-text-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
}

.import-edit :deep(textarea) {
  max-height: 65vh;
}

.text-collapse {
  width: 100%;
}

@media screen and (max-width: 767.9px) {
  .custom-text-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .custom-text-search,
  .custom-text-actions,
  .custom-text-filter-row,
  .custom-text-group-filter,
  .custom-text-search-input,
  .custom-text-group-select {
    width: 100%;
  }

  .custom-text-actions {
    justify-content: flex-start;
  }

  .custom-text-search-input :deep(.n-input),
  .custom-text-group-select :deep(.n-select) {
    width: 100%;
  }
}
</style>
