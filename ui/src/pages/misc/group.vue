<template>
  <main class="group-page">
    <header class="page-header">
      <n-card title="群组管理 / Group" :bordered="false">
        <n-grid cols="1 s:2 m:4" responsive="screen" x-gap="12" y-gap="12">
          <n-gi>
            <n-statistic label="当前结果" :value="total" />
          </n-gi>
          <n-gi>
            <n-statistic label="本页群组" :value="groups.length" />
          </n-gi>
          <n-gi>
            <n-statistic label="记录日志中" :value="loggingCount" />
          </n-gi>
          <n-gi>
            <n-statistic label="多账号群" :value="multiDiceCount" />
          </n-gi>
        </n-grid>
      </n-card>
    </header>

    <n-spin :show="listLoading">
      <section class="group-search-block">
        <ProSearchForm
          :form="groupSearchForm"
          :columns="groupSearchColumns"
          size="small"
          label-placement="left"
          label-width="96"
          cols="1 s:2 l:3 xl:5"
          :collapse-button-props="false"
        />
      </section>

      <section class="group-action-block">
        <n-tag size="small" :bordered="false" type="info">已选择 {{ selectedGroupIDs.length }} 项</n-tag>
        <n-flex size="small" align="center" class="group-meta-right">
          <n-button size="small" secondary :disabled="!selectedGroupIDs.length" @click="openBatchNotify">
            批量通知群
          </n-button>
          <n-button size="small" type="error" secondary :disabled="!selectedGroupIDs.length" @click="openBatchQuit">
            批量退群
          </n-button>
        </n-flex>
      </section>

      <section class="group-data-block">
        <FoldableCard v-for="group in groups" :key="group.groupId" class="group-card">
          <template #title>
            <n-flex align="center" size="small" wrap>
              <n-checkbox v-model:checked="group.selected" />
              <n-switch v-model:value="group.active" @update:value="markGroupChanged(group)" />
              <n-text class="group-id" tag="strong">{{ group.groupId }}</n-text>
              <n-text>「{{ group.groupName || '未获取到' }}」</n-text>
            </n-flex>
          </template>

          <template #title-extra>
            <n-button v-if="group.changed" type="success" size="small" secondary @click="saveGroup(group)">
              保存
            </n-button>
          </template>

          <template #action>
            <n-flex size="small" wrap justify="end">
              <n-button
                v-for="diceId in groupDiceIDs(group)"
                :key="diceId"
                size="small"
                type="error"
                secondary
                @click="openSingleQuit(group, diceId)"
              >
                退出 {{ diceId.slice(-4) }}
              </n-button>
            </n-flex>
          </template>

          <n-descriptions label-placement="left" size="small" :column="isMobile ? 1 : 3" bordered>
            <n-descriptions-item label="上次使用">{{ recentText(group.recentDiceSendTime) }}</n-descriptions-item>
            <n-descriptions-item label="入群时间">{{ group.enteredTime ? recentText(group.enteredTime) : '未知' }}</n-descriptions-item>
            <n-descriptions-item label="邀请人">{{ group.inviteUserId || '未知' }}</n-descriptions-item>
            <n-descriptions-item label="Log 状态">{{ group.logOn ? '开启' : '关闭' }}</n-descriptions-item>
            <n-descriptions-item label="迎新">{{ group.showGroupWelcome ? '开启' : '关闭' }}</n-descriptions-item>
            <n-descriptions-item label="群内账号">{{ groupDiceIDs(group).length || '未知' }}</n-descriptions-item>
            <n-descriptions-item label="启用扩展" :span="3">
              <n-space v-if="activeExtNames(group).length" size="small" wrap>
                <n-tag
                  v-for="ext in activeExtNames(group)"
                  :key="ext"
                  size="small"
                  :bordered="false"
                  type="success"
                >
                  {{ ext }}
                </n-tag>
              </n-space>
              <n-text v-else depth="3">未知</n-text>
            </n-descriptions-item>
          </n-descriptions>
        </FoldableCard>

        <n-empty v-if="!groups.length && !listLoading" description="暂无匹配的群组" class="group-empty" />
      </section>

      <div class="group-pagination-block">
        <n-pagination
          v-model:page="listQuery.page"
          v-model:page-size="listQuery.pageSize"
          show-size-picker
          :page-sizes="[10, 20, 30, 50]"
          :page-slot="isMobile ? 3 : 5"
          :item-count="total"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </n-spin>

    <n-modal v-model:show="notifyDialogVisible" preset="card" title="批量通知群" class="group-dialog">
      <n-flex vertical>
        <n-text depth="3">将向 {{ selectedGroupIDs.length }} 个群组发送同一条通知。</n-text>
        <n-input
          v-model:value="notifyText"
          type="textarea"
          :autosize="{ minRows: 5, maxRows: 10 }"
          placeholder="输入通知内容"
        />
        <n-flex justify="end">
          <n-button @click="notifyDialogVisible = false">取消</n-button>
          <n-button type="primary" @click="submitNotify">发送通知</n-button>
        </n-flex>
      </n-flex>
    </n-modal>

    <n-modal v-model:show="quitDialogVisible" preset="card" title="退群确认" class="group-dialog">
      <n-flex vertical>
        <n-text depth="3">
          {{ quitAction?.mode === 'single' ? '将退出当前选择的群组。' : `将批量退出 ${quitAction?.count ?? 0} 个群组。` }}
        </n-text>
        <n-checkbox v-model:checked="quitForm.silence">静默退出</n-checkbox>
        <n-checkbox v-model:checked="quitForm.saveAsDefault">保存为默认留言</n-checkbox>
        <n-input
          v-model:value="quitForm.extraText"
          type="textarea"
          :disabled="quitForm.silence"
          :autosize="{ minRows: 4, maxRows: 8 }"
          placeholder="附加留言；若静默退出则不会发送"
        />
        <n-flex justify="end">
          <n-button @click="quitDialogVisible = false">取消</n-button>
          <n-button type="error" @click="submitQuit">确认退群</n-button>
        </n-flex>
      </n-flex>
    </n-modal>
  </main>
</template>

<script setup lang="tsx">
import { computed, onMounted, reactive, ref } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useQuery } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import { useMessage } from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import {
  getSdApiV2GroupPlatformsOptions,
  postSdApiV2GroupBatchNotify,
  postSdApiV2GroupBatchQuit,
  postSdApiV2GroupList,
  postSdApiV2GroupModify,
  postSdApiV2GroupQuit,
  type GroupInfo,
} from '@/api';
import FoldableCard from '@/components/shared/FoldableCard.vue';
import { hasAccessToken } from '@/features/auth/state';
import {
  readGroupQuitDefaultText,
  writeGroupQuitDefaultText,
} from '@/features/group/quitPreference';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';

type GroupRow = GroupInfo & {
  selected?: boolean;
  changed?: boolean;
  originalActive?: boolean;
};

type QuitAction =
  | { mode: 'single'; count: 1; groupId: string; diceId: string }
  | { mode: 'batch'; count: number; groupIds: string[] };

const message = useMessage();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  platforms: [] as string[],
  queryUnusedDays: 0,
  isLogging: false,
  orderByLastTime: true,
});

type GroupSearchFormValues = {
  keyword: string;
  platforms: string[];
  queryUnusedDays: number;
  isLogging: boolean;
  orderByLastTime: boolean;
};

const defaultGroupSearchFormValues = (): GroupSearchFormValues => ({
  keyword: '',
  platforms: [],
  queryUnusedDays: 0,
  isLogging: false,
  orderByLastTime: true,
});

const groups = ref<GroupRow[]>([]);
const total = ref(0);
const listLoading = ref(false);

const notifyDialogVisible = ref(false);
const notifyText = ref('');

const quitDialogVisible = ref(false);
const quitAction = ref<QuitAction | null>(null);
const quitForm = reactive({
  silence: false,
  extraText: readGroupQuitDefaultText(),
  saveAsDefault: false,
});

const platformsQuery = useQuery({
  ...getSdApiV2GroupPlatformsOptions(),
  enabled: hasAccessToken,
});

const platformOptions = computed(() =>
  (platformsQuery.data.value?.item ?? []).map(item => ({
    label: item.text,
    value: item.value,
  })),
);

const groupSearchForm = createProSearchForm<GroupSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultGroupSearchFormValues()),
  onSubmit: async values => {
    Object.assign(listQuery, {
      ...values,
      page: 1,
    });
    await searchGroups();
  },
  onReset: async () => {
    Object.assign(listQuery, {
      ...defaultGroupSearchFormValues(),
      page: 1,
    });
    await searchGroups();
  },
});

const groupSearchColumns = computed<ProSearchFormColumns<GroupSearchFormValues>>(() => [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索群号 / 群名',
    },
  },
  {
    label: '平台',
    path: 'platforms',
    field: 'select',
    fieldProps: {
      options: platformOptions.value,
      multiple: true,
      clearable: true,
      placeholder: '平台筛选',
    },
  },
  {
    label: '未使用天数',
    path: 'queryUnusedDays',
    field: 'digit',
    fieldProps: {
      min: 0,
      placeholder: '未使用天数',
    },
  },
  {
    label: '仅记录日志',
    path: 'isLogging',
    field: 'switch',
  },
  {
    label: '按最后使用排序',
    path: 'orderByLastTime',
    field: 'switch',
  },
]);

const selectedGroups = computed(() => groups.value.filter(item => item.selected));
const selectedGroupIDs = computed(() => selectedGroups.value.map(item => item.groupId));
const loggingCount = computed(() => groups.value.filter(item => item.logOn).length);
const multiDiceCount = computed(() => groups.value.filter(item => groupDiceIDs(item).length > 1).length);

function groupDiceIDs(group: GroupInfo): string[] {
  return Object.keys((group.diceIdExistsMap ?? {}) as Record<string, unknown>).sort();
}

function activeExtNames(group: GroupInfo): string[] {
  return group.tmpExtList ?? [];
}

function recentText(timestamp: number): string {
  return timestamp ? dayjs.unix(timestamp).fromNow() : '从未';
}

async function searchGroups() {
  listLoading.value = true;
  try {
    const { data } = await postSdApiV2GroupList({
      body: {
        page: listQuery.page,
        pageSize: listQuery.pageSize,
        keyword: listQuery.keyword || '',
        filter: {
          platforms: listQuery.platforms.length ? listQuery.platforms : undefined,
          orderByLastTime: listQuery.orderByLastTime,
          queryUnusedDays: listQuery.queryUnusedDays || undefined,
          isLogging: listQuery.isLogging || undefined,
        },
      },
      throwOnError: true,
    });

    groups.value = (data.item.list ?? []).map(item => ({
      ...item,
      selected: false,
      changed: false,
      originalActive: item.active,
    }));
    total.value = Number(data.item.total ?? 0);
  } finally {
    listLoading.value = false;
  }
}

function handlePageChange(value: number) {
  listQuery.page = value;
  void searchGroups();
}

function handlePageSizeChange(value: number) {
  listQuery.page = 1;
  listQuery.pageSize = value;
  void searchGroups();
}

function markGroupChanged(group: GroupRow) {
  group.changed = group.originalActive !== group.active;
}

async function saveGroup(group: GroupRow) {
  const { data } = await postSdApiV2GroupModify({
    body: {
      active: group.active,
      groupId: group.groupId,
    },
    throwOnError: true,
  });
  if (data.message !== 'ok') {
    message.error(data.message || '保存失败');
    return;
  }
  group.originalActive = group.active;
  group.changed = false;
  message.success('群服务状态已保存');
}

function openSingleQuit(group: GroupRow, diceId: string) {
  quitAction.value = {
    mode: 'single',
    count: 1,
    groupId: group.groupId,
    diceId,
  };
  quitDialogVisible.value = true;
}

function openBatchQuit() {
  if (!selectedGroupIDs.value.length) {
    message.warning('请先选择群组');
    return;
  }
  quitAction.value = {
    mode: 'batch',
    count: selectedGroupIDs.value.length,
    groupIds: [...selectedGroupIDs.value],
  };
  quitDialogVisible.value = true;
}

async function submitQuit() {
  if (!quitAction.value) return;

  if (quitAction.value.mode === 'single') {
    const { data } = await postSdApiV2GroupQuit({
      body: {
        groupId: quitAction.value.groupId,
        diceId: quitAction.value.diceId,
        silence: quitForm.silence,
        extraText: quitForm.extraText,
      },
      throwOnError: true,
    });
    if (data.message !== 'ok') {
      message.error(data.message || '退群失败');
      return;
    }
    message.success('退群完成');
  } else {
    const { data } = await postSdApiV2GroupBatchQuit({
      body: {
        groupIds: quitAction.value.groupIds,
        silence: quitForm.silence,
        extraText: quitForm.extraText,
      },
      throwOnError: true,
    });
    message.success(`批量退群完成，共处理 ${data.item} 个群组`);
  }

  if (quitForm.saveAsDefault) {
    writeGroupQuitDefaultText(quitForm.extraText);
  }
  quitDialogVisible.value = false;
  await searchGroups();
}

function openBatchNotify() {
  if (!selectedGroupIDs.value.length) {
    message.warning('请先选择群组');
    return;
  }
  notifyDialogVisible.value = true;
}

async function submitNotify() {
  const text = notifyText.value.trim();
  if (!text) {
    message.warning('请输入通知内容');
    return;
  }
  const { data } = await postSdApiV2GroupBatchNotify({
    body: {
      groupIds: selectedGroupIDs.value,
      text,
    },
    throwOnError: true,
  });
  notifyDialogVisible.value = false;
  notifyText.value = '';
  message.success(`已通知 ${data.item} 个群组`);
}

onMounted(async () => {
  await searchGroups();
});
</script>

<style scoped>
.page-header {
  margin-bottom: 1rem;
}

.group-search-block {
  margin-bottom: 1rem;
}

.tool-label {
  margin-right: 0.5rem;
}

.group-action-block {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.group-meta-right {
  margin-left: auto;
}

.group-data-block {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.group-card {
  width: 100%;
}

.group-id {
  max-width: 24rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-pagination-block {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.group-empty {
  padding: 2rem 0;
}

.group-dialog {
  width: min(720px, calc(100vw - 2rem));
}

@media screen and (max-width: 700px) {
  .group-action-block {
    flex-direction: column;
    align-items: flex-start;
  }

  .group-meta-right {
    margin-left: 0;
  }

  .group-id {
    max-width: 100%;
  }

  .group-pagination-block {
    justify-content: flex-start;
  }
}
</style>
