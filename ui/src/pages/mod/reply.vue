<template>
  <main class="reply-page">
    <n-spin :show="pageBusy">
      <header class="page-head">
        <n-flex align="center" justify="space-between" wrap>
          <n-switch
            :value="replyEnabled"
            :loading="replyConfigMutation.isPending.value"
            @update:value="handleReplySwitchUpdate"
          >
            <template #checked>启用</template>
            <template #unchecked>关闭</template>
            总开关
          </n-switch>

          <n-button
            v-if="replyEnabled"
            type="primary"
            :loading="saveMutation.isPending.value"
            :disabled="!modified || !currentFileDraft"
            @click="saveCurrent"
          >
            <template #icon>
              <n-icon><i-carbon-save /></n-icon>
            </template>
            保存
          </n-button>
        </n-flex>
      </header>

      <template v-if="!replyEnabled">
        <section class="reply-empty">
          <n-text type="error" class="text-xl">请先启用总开关！</n-text>
        </section>
      </template>

      <template v-else>
        <section class="reply-layout">
          <aside class="reply-sidebar">
            <div class="panel-head">
              <div class="panel-title">
                <n-icon><i-carbon-folder /></n-icon>
                <span>文件管理</span>
              </div>
              <div class="panel-toolbar">
                <n-tooltip>
                  <template #trigger>
                    <n-button size="small" quaternary circle @click="newFileDialogVisible = true">
                      <template #icon>
                        <n-icon><i-carbon-document-add /></n-icon>
                      </template>
                    </n-button>
                  </template>
                  新建文件
                </n-tooltip>

                <n-tooltip>
                  <template #trigger>
                    <n-upload
                      action=""
                      accept=".yaml"
                      :custom-request="uploadFile"
                      :show-file-list="false"
                    >
                      <n-button size="small" quaternary circle>
                        <template #icon>
                          <n-icon><i-carbon-upload /></n-icon>
                        </template>
                      </n-button>
                    </n-upload>
                  </template>
                  上传文件
                </n-tooltip>

                <n-tooltip>
                  <template #trigger>
                    <n-button size="small" quaternary circle @click="importDialogVisible = true">
                      <template #icon>
                        <n-icon><i-carbon-document /></n-icon>
                      </template>
                    </n-button>
                  </template>
                  解析导入
                </n-tooltip>

                <n-tooltip>
                  <template #trigger>
                    <n-button size="small" quaternary circle :disabled="!selectedFilename" @click="downloadCurrentFile">
                      <template #icon>
                        <n-icon><i-carbon-download /></n-icon>
                      </template>
                    </n-button>
                  </template>
                  下载文件
                </n-tooltip>

                <n-tooltip>
                  <template #trigger>
                    <n-button size="small" quaternary circle type="error" :disabled="!selectedFilename" @click="deleteCurrentFile">
                      <template #icon>
                        <n-icon><i-carbon-row-delete /></n-icon>
                      </template>
                    </n-button>
                  </template>
                  删除文件
                </n-tooltip>
              </div>
            </div>

            <div class="panel-controls">
              <n-input
                v-model:value="fileQuery.keyword"
                clearable
                size="small"
                placeholder="按文件名搜索"
              >
                <template #prefix>
                  <n-icon><i-carbon-search /></n-icon>
                </template>
              </n-input>

              <div class="sort-row">
                <n-select
                  v-model:value="fileQuery.sortBy"
                  size="small"
                  :options="fileSortOptions"
                  class="sort-select"
                />
                <n-select
                  v-model:value="fileQuery.sortOrder"
                  size="small"
                  :options="fileSortOrderOptions"
                  class="order-select"
                />
              </div>
            </div>

            <div class="panel-body">
              <n-empty v-if="!fileItems.length" description="暂无文件" />
              <button
                v-for="item in fileItems"
                :key="item.filename"
                type="button"
                class="file-item"
                :class="{ active: item.filename === selectedFilename }"
                @click="selectFile(item.filename)"
              >
                <div class="file-item-main">
                  <div class="file-item-name-row">
                    <n-icon class="file-item-icon"><i-carbon-document-blank /></n-icon>
                    <span class="file-item-name">{{ item.filename }}</span>
                  </div>
                  <div class="file-item-meta">
                    <n-tag size="tiny" :bordered="false" :type="getFileEnableStatus(item.filename, item.enable) ? 'success' : 'warning'">
                      {{ getFileEnableStatus(item.filename, item.enable) ? '启用' : '停用' }}
                    </n-tag>
                    <span>{{ formatUpdateTime(item.updateTimestamp) }}</span>
                    <span>{{ item.itemCount }} 条</span>
                  </div>
                </div>
              </button>
            </div>

            <div class="panel-footer">
              <n-pagination
                v-model:page="fileQuery.page"
                :page-size="fileQuery.pageSize"
                :item-count="fileTotal"
                simple
              />
            </div>
          </aside>

          <section class="reply-content">
            <n-empty v-if="!selectedFilename || !currentFileDraft" description="请选择一个文件" />

            <template v-else>
              <section class="reply-section">
                <div class="section-head">
                  <div>
                    <h3>前置条件</h3>
                    <p>该文件下所有规则执行前都需要先满足这些条件。</p>
                  </div>
                  <n-space size="small" align="center">
                    <n-button
                      size="small"
                      :type="currentFileDraft.enable ? 'success' : 'warning'"
                      secondary
                      @click="toggleCurrentFileEnable"
                    >
                      <template #icon>
                        <n-icon>
                          <i-carbon-checkmark-filled v-if="currentFileDraft.enable" />
                          <i-carbon-close-outline v-else />
                        </n-icon>
                      </template>
                      {{ currentFileDraft.enable ? '文件已启用' : '文件未启用' }}
                    </n-button>
                    <n-button size="small" secondary @click="addCommonCondition">
                      <template #icon>
                        <n-icon><i-carbon-add-large /></n-icon>
                      </template>
                      添加条件
                    </n-button>
                  </n-space>
                </div>

                <div class="section-body">
                  <ConditionBuilder
                    v-if="pagedCommonConditions.length"
                    v-model="pagedCommonConditions"
                    @delete-condition="deleteCommonCondition"
                  />
                  <n-empty v-else description="当前无前置条件" size="small" />
                </div>

                <div class="section-footer">
                  <n-pagination
                    v-model:page="commonConditionsPage"
                    :page-size="commonConditionsPageSize"
                    :item-count="commonConditionsTotal"
                    simple
                  />
                </div>
              </section>

              <section class="reply-section">
                <div class="section-head">
                  <div>
                    <h3>规则列表</h3>
                    <p>从上到下匹配。当前页只显示部分规则，但保存会提交整份文件。</p>
                  </div>
                  <n-space size="small" align="center">
                    <n-button size="small" secondary @click="addReplyItem">
                      <template #icon>
                        <n-icon><i-carbon-add-large /></n-icon>
                      </template>
                      添加规则
                    </n-button>
                  </n-space>
                </div>

                <div class="section-body">
                  <NestedRuleEditor
                    :tasks="rulePageItems"
                    :start-index="rulePageStart"
                    @change="markModified"
                    @delete-rule="deleteReplyItem"
                  />
                </div>

                <div class="section-footer">
                  <n-pagination
                    v-model:page="rulesPage"
                    :page-size="rulesPageSize"
                    :item-count="rulesTotal"
                    simple
                  />
                </div>
              </section>
            </template>
          </section>
        </section>
      </template>

      <n-modal
        v-model:show="importDialogVisible"
        preset="card"
        title="导入配置"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        :show-close="false"
        class="the-dialog"
      >
        <n-input
          v-model:value="configForImport"
          placeholder="支持格式: 关键字/回复语"
          class="reply-text"
          type="textarea"
          :autosize="{ minRows: 4, maxRows: 10 }"
        />
        <template #footer>
          <n-flex>
            <n-button @click="importDialogVisible = false">取消</n-button>
            <n-button type="primary" :disabled="configForImport === '' || !currentFileDraft" @click="doImport">
              下一步
            </n-button>
          </n-flex>
        </template>
      </n-modal>

      <n-modal
        v-model:show="licenseDialogVisible"
        preset="card"
        title="许可协议"
        :mask-closable="false"
        :close-on-esc="false"
        :closable="false"
        class="the-dialog"
      >
        <pre class="license-text">尊敬的用户，欢迎您选择由木落等研发的海豹骰点核心（SealDice），在您使用自定义功能前，请务必仔细阅读使用须知，当您使用我们提供的服务时，即代表您已同意使用须知的内容。

  您需了解，海豹核心官方版只支持 TRPG 功能，娱乐功能定制化请自便，和海豹无关。
  您清楚并明白您对通过骰子提供的全部内容负责，包括自定义回复、非自带的插件、牌堆。海豹骰不对非自身提供以外的内容合法性负责。您不得在使用海豹骰服务时，导入包括但不限于以下情形的内容：
  (1) 反对中华人民共和国宪法所确定的基本原则的；
  (2) 危害国家安全，泄露国家秘密，颠覆国家政权，破坏国家统一的;
  (3) 损害国家荣誉和利益的;
  (4) 煽动民族仇恨、民族歧视、破坏民族团结的；
  (5) 破坏国家宗教政策，宣扬邪教和封建迷信的;
  (6) 散布谣言，扰乱社会秩序，破坏社会稳定的；
  (7) 散布淫秽、色情、赌博、暴力、凶杀、恐怖或者教唆犯罪的；
  (8) 侮辱或者诽谤他人，侵害他人合法权益的；
  (9) 宣扬、教唆使用外挂、私服、病毒、恶意代码、木马及其相关内容的；
  (10) 侵犯他人知识产权或涉及第三方商业秘密及其他专有权利的；
  (11) 散布任何贬损、诋毁、恶意攻击海豹骰及开发人员、海洋馆工作人员、mod 编写人员、关联合作者的；
  (12) 含有中华人民共和国法律、行政法规、政策、上级主管部门下发通知中所禁止的其他内容的。

  一旦查实您有以上禁止行为，请立即停用海豹骰。同时我们也会主动对你进行举报。</pre>
        <template #footer>
          <n-flex>
            <n-button type="primary" :loading="replyConfigMutation.isPending.value" @click="acceptLicense">
              我同意
            </n-button>
            <n-button type="error" @click="refuseLicense">我拒绝</n-button>
          </n-flex>
        </template>
      </n-modal>

      <n-modal
        v-model:show="newFileDialogVisible"
        preset="dialog"
        title="创建一个新的回复文件"
        positive-text="确定"
        negative-text="取消"
        @positive-click="createNewFile"
      >
        <n-input v-model:value="newFilename" placeholder="reply2.yaml" />
      </n-modal>
    </n-spin>
  </main>
</template>

<script setup lang="ts">
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
  type FileInfo,
  type ReplyFileDetail,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import ConditionBuilder from '@/components/shared/ConditionBuilder.vue';
import NestedRuleEditor from '@/components/shared/NestedRuleEditor.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import { useUnsavedChanges } from '@/features/unsavedChanges';

// 自定义回复页是本项目最复杂的表单页之一。
// 设计上将“远端文件详情”和“本地草稿”分开：远端数据来自 Query，
// 草稿存放在 drafts 中，用户保存前所有编辑都只影响草稿，modified 负责提示未保存状态。
type ReplyCondition = {
  condType: string;
  matchType: string;
  matchOp?: string;
  value: string | number;
};

type ReplyMessage = [string, number];

type ReplyResult = {
  resultType: string;
  delay: number;
  message: ReplyMessage[];
};

type ReplyTask = {
  enable: boolean;
  conditions: ReplyCondition[];
  results: ReplyResult[];
};

type ReplyFileDraft = {
  enable: boolean;
  interval: number;
  name: string;
  author: string[];
  version: string;
  createTimestamp: number;
  updateTimestamp: number;
  desc: string;
  storeID: string;
  conditions: ReplyCondition[];
  items: ReplyTask[];
  filename: string;
  itemCount: number;
};

type ReplyApiCondition = {
  condType?: string;
  matchType?: string;
  matchOp?: string;
  value?: unknown;
};

type ReplyApiResult = {
  resultType?: string;
  delay?: unknown;
  message?: unknown[] | null;
};

type ReplyApiTask = {
  enable?: boolean;
  conditions?: unknown[] | null;
  results?: unknown[] | null;
};

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

const fileQuery = reactive({
  page: 1,
  pageSize: 30,
  keyword: '',
  sortBy: 'updateTime',
  sortOrder: 'desc',
});

const fileSortOptions = [
  { label: '按更新时间', value: 'updateTime' },
  { label: '按名称', value: 'name' },
];

const fileSortOrderOptions = [
  { label: '降序', value: 'desc' },
  { label: '升序', value: 'asc' },
];

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
    const start = (pageData.page - 1) * pageData.pageSize;
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

const replyConfigMutation = useMutation({
  mutationFn: async (enabled: boolean) => {
    const { data } = await putSdApiV2ConfigReply({
      body: { body: { customReplyConfigEnable: enabled } },
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
      body: { body: toApiReplyConfig(currentFileDraft.value) },
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
      body: { body: { filename: target } },
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
      parseAs: 'blob',
      throwOnError: true,
    }),
    selectedFilename.value,
  );
}

async function uploadFile(options: UploadCustomRequestOptions) {
  try {
    await postSdApiV2CustomReplyFilesUpload({
      body: { file: options.file.file as File },
      headers: { 'Content-Type': undefined as never },
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

function parseString(str: string): [string[], string[], string] {
  const leftArr: string[] = [];
  const rightArr: string[] = [];
  let restIndex = 0;

  let currentStr = '';
  let isLeft = true;
  let isEscaped = false;

  for (let i = 0; i < str.length; i++) {
    const char = str[i];
    restIndex = i;
    if (isEscaped) {
      if (char !== '\r' && char !== '\n' && char !== '/') {
        currentStr += '\\';
      }
      if (char === 'n' || char === 'r') {
        currentStr = currentStr.slice(0, -1) + '\n';
      } else {
        currentStr += char;
      }
      isEscaped = false;
      continue;
    }
    if (char === '\n') break;
    if (char === '\\') {
      isEscaped = true;
      continue;
    }
    if (char === '|') {
      if (isLeft) leftArr.push(currentStr);
      else rightArr.push(currentStr);
      currentStr = '';
    } else if (char === '/') {
      if (i < str.length - 1 && str[i + 1] === '|') {
        currentStr += char;
      } else {
        if (isLeft) leftArr.push(currentStr);
        else rightArr.push(currentStr);
        currentStr = '';
        isLeft = false;
      }
    } else {
      currentStr += char;
    }
  }
  if (isLeft) leftArr.push(currentStr);
  else rightArr.push(currentStr);
  return [leftArr, rightArr, str.slice(restIndex + 1)];
}

function doImport() {
  if (!currentFileDraft.value) return;
  let text = configForImport.value;
  while (true) {
    const [conditions, replies, rest] = parseString(text);
    if (conditions.length && replies.length) {
      currentFileDraft.value.items.push({
        enable: true,
        conditions: [{
          condType: 'textMatch',
          matchType: 'matchMulti',
          value: conditions.join('|'),
        }],
        results: [{
          resultType: 'replyToSender',
          delay: 0,
          message: replies.map(reply => [reply, 1]),
        }],
      });
    }
    text = rest;
    if (!rest) break;
  }
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

function cloneReplyTask(item: ReplyTask): ReplyTask {
  return {
    enable: item.enable,
    conditions: item.conditions.map(condition => ({ ...condition })),
    results: item.results.map(result => ({
      resultType: result.resultType,
      delay: result.delay,
      message: result.message.map(messageItem => [messageItem[0], messageItem[1]]),
    })),
  };
}

function cloneReplyFileDraft(item: ReplyFileDraft): ReplyFileDraft {
  return {
    ...item,
    author: [...item.author],
    conditions: item.conditions.map(condition => ({ ...condition })),
    items: item.items.map(task => cloneReplyTask(task)),
  };
}

function normalizeReplyFileDetail(detail: ReplyFileDetail): ReplyFileDraft {
  return {
    enable: detail.enable,
    interval: Number(detail.interval ?? 0),
    name: String(detail.name ?? ''),
    author: (detail.author ?? []).map(item => String(item)),
    version: String(detail.version ?? ''),
    createTimestamp: Number(detail.createTimestamp ?? 0),
    updateTimestamp: Number(detail.updateTimestamp ?? 0),
    desc: String(detail.desc ?? ''),
    storeID: String(detail.storeID ?? ''),
    conditions: normalizeConditions(detail.conditions as unknown[] | null | undefined),
    items: [],
    filename: String(detail.filename ?? ''),
    itemCount: Number(detail.itemCount ?? 0),
  };
}

function normalizeReplyTask(item: unknown): ReplyTask {
  const raw = (item ?? {}) as ReplyApiTask;
  return {
    enable: raw.enable !== false,
    conditions: normalizeConditions(raw.conditions),
    results: normalizeResults(raw.results),
  };
}

function normalizeConditions(items: unknown[] | null | undefined): ReplyCondition[] {
  return (items ?? []).map(item => normalizeCondition(item));
}

function normalizeCondition(item: unknown): ReplyCondition {
  const raw = (item ?? {}) as ReplyApiCondition;
  return {
    condType: String(raw.condType ?? 'textMatch'),
    matchType: String(raw.matchType ?? 'matchExact'),
    ...(raw.matchOp ? { matchOp: String(raw.matchOp) } : {}),
    value: typeof raw.value === 'number' ? raw.value : String(raw.value ?? ''),
  };
}

function normalizeResults(items: unknown[] | null | undefined): ReplyResult[] {
  return (items ?? []).map(item => {
    const raw = (item ?? {}) as ReplyApiResult;
    return {
      resultType: String(raw.resultType ?? 'replyToSender'),
      delay: Number(raw.delay ?? 0),
      message: normalizeMessages(raw.message),
    };
  });
}

function normalizeMessages(items: unknown[] | null | undefined): ReplyMessage[] {
  return (items ?? []).map(item => {
    const parts = Array.isArray(item) ? item : [];
    return [String(parts[0] ?? ''), Number(parts[1] ?? 1)];
  });
}

function toApiReplyConfig(draft: ReplyFileDraft) {
  return {
    enable: draft.enable,
    interval: Number(draft.interval) || 0,
    items: draft.items.filter(Boolean).map(item => ({
      enable: item.enable,
      conditions: item.conditions.map(condition => {
        if (condition.condType === 'textLenLimit') {
          return {
            ...condition,
            value: Number(condition.value) || 0,
          };
        }
        return condition;
      }),
      results: item.results.map(result => ({
        ...result,
        delay: Number(result.delay) || 0,
      })),
    })),
    name: draft.name,
    author: draft.author,
    version: draft.version,
    createTimestamp: draft.createTimestamp,
    updateTimestamp: draft.updateTimestamp,
    desc: draft.desc,
    storeID: draft.storeID,
    filename: draft.filename,
    conditions: draft.conditions.map(condition => {
      if (condition.condType === 'textLenLimit') {
        return {
          ...condition,
          value: Number(condition.value) || 0,
        };
      }
      return condition;
    }),
  };
}
</script>

<style scoped>
.reply-page {
  width: 100%;
  max-width: none;
  margin: 0 auto;
  text-align: left;
}

.page-head {
  margin-bottom: 1rem;
}

.reply-empty {
  padding: 2rem 0;
}

.reply-layout {
  display: flex;
  min-width: 0;
  height: calc(100vh - 178px);
  min-height: 620px;
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.reply-sidebar {
  display: flex;
  width: 280px;
  min-width: 240px;
  max-width: 320px;
  min-height: 0;
  flex-direction: column;
  border-right: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated-muted);
}

@supports (color: color-mix(in srgb, white, black)) {
  .reply-sidebar {
    background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 48%);
  }
}

.panel-head,
.panel-controls,
.panel-footer {
  border-bottom: 1px solid var(--sd-border-soft);
  padding: 0.75rem;
}

.panel-footer {
  border-top: 1px solid var(--sd-border-soft);
  border-bottom: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 700;
  line-height: 1;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-top: 0.5rem;
}

.panel-controls {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.sort-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 72px;
  gap: 0.5rem;
}

.sort-select,
.order-select {
  min-width: 0;
}

.panel-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 0.5rem;
}

.file-item {
  display: block;
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  padding: 0.45rem 0.5rem;
  text-align: left;
}

.file-item:hover {
  background: var(--sd-bg-hover);
}

.file-item.active {
  background: var(--sd-bg-selected);
}

.file-item-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.35rem;
}

.file-item-name-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
}

.file-item-icon {
  flex: 0 0 auto;
}

.file-item-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
}

.reply-content {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: auto;
  background: var(--sd-bg-page);
}

.reply-section {
  flex: 0 0 auto;
  border: 0;
  border-bottom: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
}

.reply-section:last-child {
  flex: 1 1 auto;
  min-height: 0;
  border-bottom: 0;
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--sd-border-soft);
  padding: 1rem;
}

.section-head h3 {
  margin: 0;
  font-size: 1rem;
}

.section-head p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}

.section-body {
  padding: 1rem;
  min-width: 0;
}

.section-footer {
  border-top: 1px solid var(--sd-border-soft);
  padding: 0.75rem 1rem;
}

.reply-text :deep(textarea) {
  max-height: 65vh;
}

.license-text {
  white-space: pre-wrap;
}

@media screen and (max-width: 1023.9px) {
  .reply-layout {
    height: calc(100vh - 148px);
    min-height: 560px;
  }

  .reply-sidebar {
    width: 240px;
    min-width: 220px;
  }

  .section-head {
    flex-direction: column;
  }
}

@media screen and (max-width: 639.9px) {
  .reply-layout {
    height: auto;
    min-height: 0;
    flex-direction: column;
  }

  .reply-sidebar {
    width: 100%;
    max-width: none;
    border-right: 0;
    border-bottom: 1px solid var(--sd-border);
  }

  .panel-body {
    max-height: 220px;
  }
}
</style>
