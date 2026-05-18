<script setup lang="ts">
import { computed, ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  NButton,
  NFlex,
  NInput,
  NModal,
  NPagination,
  NSelect,
  NStatistic,
  NText,
  useDialog,
  useMessage,
  NAlert,
} from 'naive-ui';
import {
  getSdApiV2JsList,
  getSdApiV2JsByNameDataList,
  getSdApiV2JsByNameData,
  getSdApiV2JsByNameDataInfo,
  postSdApiV2JsByNameData,
  postSdApiV2JsByNameDataDelete,
  postSdApiV2JsByNameDataShrink,
  type JsInfo,
} from '@/api';
import { hasAccessToken } from '@/features/auth/state';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

const selectedPlugin = ref<string>('');

const pluginListQuery = useQuery({
  queryKey: ['js-list-for-data'],
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2JsList({
      query: { page: 1, pageSize: 200 },
      throwOnError: true,
    });
    return data.item.data ?? [];
  },
});

const pluginOptions = computed(() =>
  (pluginListQuery.data.value ?? []).map((p: JsInfo) => ({
    label: `${p.name} (v${p.version})`,
    value: p.name,
  })),
);

const dataPage = ref({ page: 1, pageSize: 20 });
const dataKeyword = ref('');

const dataListQuery = useQuery({
  queryKey: computed(() => ['js-data-list', selectedPlugin.value, dataPage.value, dataKeyword.value]),
  enabled: computed(() => hasAccessToken.value && !!selectedPlugin.value),
  queryFn: async () => {
    const { data } = await getSdApiV2JsByNameDataList({
      path: { name: selectedPlugin.value },
      query: { page: dataPage.value.page, pageSize: dataPage.value.pageSize, keyword: dataKeyword.value || undefined },
      throwOnError: true,
    });
    return data.item;
  },
});

const dataInfoQuery = useQuery({
  queryKey: computed(() => ['js-data-info', selectedPlugin.value]),
  enabled: computed(() => hasAccessToken.value && !!selectedPlugin.value),
  queryFn: async () => {
    const { data } = await getSdApiV2JsByNameDataInfo({
      path: { name: selectedPlugin.value },
      throwOnError: true,
    });
    return data.item;
  },
});

// Edit modal
const showEditModal = ref(false);
const editKey = ref('');
const editValue = ref('');
const editIsJson = ref(false);
const jsonError = ref('');

function openEdit(key: string, value: string, isJson: boolean) {
  editKey.value = key;
  editValue.value = value;
  editIsJson.value = isJson;
  jsonError.value = '';
  try {
    JSON.parse(value);
  } catch {
    if (isJson) {
      jsonError.value = '数据格式不是合法 JSON';
    }
  }
  showEditModal.value = true;
}

const setMutation = useMutation({
  mutationFn: async (payload: { key: string; value: string }) => {
    await postSdApiV2JsByNameData({
      path: { name: selectedPlugin.value },
      body: { body: payload },
      throwOnError: true,
    });
  },
  onSuccess: () => {
    message.success('已保存');
    showEditModal.value = false;
    invalidateData();
  },
  onError: () => message.error('保存失败'),
});

const deleteMutation = useMutation({
  mutationFn: async (keys: string[]) => {
    await postSdApiV2JsByNameDataDelete({
      path: { name: selectedPlugin.value },
      body: { body: { keys } },
      throwOnError: true,
    });
  },
  onSuccess: () => {
    message.success('已删除');
    invalidateData();
  },
  onError: () => message.error('删除失败'),
});

const shrinkMutation = useMutation({
  mutationFn: async () => {
    await postSdApiV2JsByNameDataShrink({
      path: { name: selectedPlugin.value },
      throwOnError: true,
    });
  },
  onSuccess: () => {
    message.success('数据库已压缩');
    queryClient.invalidateQueries({ queryKey: ['js-data-info'] });
  },
  onError: () => message.error('压缩失败'),
});

function invalidateData() {
  queryClient.invalidateQueries({ queryKey: ['js-data-list'] });
  queryClient.invalidateQueries({ queryKey: ['js-data-info'] });
}

function confirmSave() {
  if (editIsJson.value && jsonError.value) {
    dialog.warning({
      title: 'JSON 格式错误',
      content: '当前数据不是合法 JSON，确定要继续保存吗？',
      positiveText: '仍然保存',
      negativeText: '取消',
      onPositiveClick: () => setMutation.mutateAsync({ key: editKey.value, value: editValue.value }),
    });
    return;
  }
  setMutation.mutateAsync({ key: editKey.value, value: editValue.value });
}

function handleDeleteKey(key: string) {
  dialog.warning({
    title: '删除 Key',
    content: `确认删除 "${key}"？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => deleteMutation.mutateAsync([key]),
  });
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
</script>

<template>
  <div>
    <header class="mb-4">
      <n-flex size="small" align="center">
        <n-text>选择插件：</n-text>
        <n-select
          v-model:value="selectedPlugin"
          :options="pluginOptions"
          placeholder="选择插件"
          class="w-60"
          clearable
        />
      </n-flex>
    </header>

    <template v-if="selectedPlugin">
      <!-- Info -->
      <n-flex v-if="dataInfoQuery.data.value" size="medium" class="mb-4">
        <n-statistic label="Key 数量" :value="dataInfoQuery.data.value.keyCount" />
        <n-statistic label="文件大小" :value="formatFileSize(dataInfoQuery.data.value.fileSize)" />
        <n-button
          v-if="dataInfoQuery.data.value.canShrink"
          size="small"
          :loading="shrinkMutation.isPending.value"
          @click="shrinkMutation.mutateAsync()"
        >
          压缩数据库
        </n-button>
      </n-flex>

      <!-- Search -->
      <n-flex size="small" class="mb-4">
        <n-input
          v-model:value="dataKeyword"
          placeholder="搜索 Key（支持 * ? 通配符）"
          clearable
          class="w-80"
          @keyup.enter="invalidateData()"
        />
        <n-button size="small" @click="invalidateData()">查询</n-button>
      </n-flex>

      <!-- List -->
      <section class="data-list">
        <div
          v-for="kv in dataListQuery.data.value?.keys ?? []"
          :key="kv.key"
          class="data-row"
        >
          <n-flex align="center" justify="space-between" wrap>
            <n-flex align="center" size="small">
              <n-text class="data-key">{{ kv.key }}</n-text>
              <n-tag v-if="kv.isJson" size="small" type="info" :bordered="false">JSON</n-tag>
            </n-flex>
            <n-flex size="small">
              <n-button size="tiny" @click="openEdit(kv.key, kv.value, kv.isJson)">编辑</n-button>
              <n-button size="tiny" type="error" secondary @click="handleDeleteKey(kv.key)">删除</n-button>
            </n-flex>
          </n-flex>
        </div>
        <n-text v-if="!dataListQuery.data.value?.keys?.length" depth="3">无数据</n-text>
      </section>

      <!-- Pagination -->
      <div v-if="(dataListQuery.data.value?.total ?? 0) > dataPage.pageSize" class="js-data-pagination">
        <n-pagination
          v-model:page="dataPage.page"
          :page-size="dataPage.pageSize"
          :item-count="dataListQuery.data.value?.total ?? 0"
          :page-slot="3"
        />
      </div>
    </template>

    <!-- Edit Modal -->
    <n-modal v-model:show="showEditModal" title="编辑数据" preset="card" style="width: 90vw; max-width: 700px">
      <n-flex vertical size="medium">
        <n-flex align="center">
          <n-text depth="3" class="w-16">Key:</n-text>
          <n-text>{{ editKey }}</n-text>
        </n-flex>
        <n-flex align="center" v-if="editIsJson">
          <n-text depth="3" class="w-16">格式:</n-text>
          <n-tag size="small" type="info" :bordered="false">JSON</n-tag>
        </n-flex>
        <n-flex vertical>
          <n-text depth="3">Value:</n-text>
          <n-input
            v-model:value="editValue"
            type="textarea"
            rows="10"
            @input="() => {
              jsonError = '';
              try { JSON.parse(editValue); } catch { if (editIsJson) jsonError = 'JSON 格式不合法'; }
            }"
          />
        </n-flex>
        <n-alert v-if="jsonError" type="warning" :show-icon="true">
          {{ jsonError }}
        </n-alert>
      </n-flex>
      <template #footer>
        <n-flex justify="end">
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button
            type="primary"
            :loading="setMutation.isPending.value"
            @click="confirmSave"
          >
            保存
          </n-button>
        </n-flex>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.data-row {
  padding: 0.4rem 0.5rem;
}
.data-row + .data-row {
  border-top: 1px solid var(--sd-border-soft);
}
.data-key {
  font-family: monospace;
  font-size: 0.9rem;
  word-break: break-all;
}
.js-data-pagination {
  display: flex;
  justify-content: center;
  margin-top: 1rem;
}
.w-16 {
  width: 4rem;
}
.w-60 {
  width: 15rem;
}
.w-80 {
  width: 20rem;
}
.mb-4 {
  margin-bottom: 1rem;
}
</style>
