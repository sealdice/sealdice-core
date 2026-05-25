<template>
  <div>
    <header class="mb-4">
      <ProSearchForm
        :form="searchForm"
        :columns="searchColumns"
        size="small"
        label-placement="left"
        label-width="72"
        cols="1 s:2"
        :collapse-button-props="false"
      />
    </header>

    <template v-if="selectedPlugin">
      <!-- Info -->
      <n-flex v-if="dataInfoQuery.data.value" size="medium" class="mb-4" wrap>
        <n-statistic label="Key 数量" :value="dataInfoQuery.data.value.keyCount" />
        <n-statistic label="文件大小" :value="formatFileSize(dataInfoQuery.data.value.fileSize)" />
        <n-button
          v-if="dataInfoQuery.data.value.canShrink"
          size="small"
          :loading="shrinkMutation.isPending.value"
          @click="shrinkData"
        >
          压缩数据库
        </n-button>
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

<script setup lang="ts">
import { computed, ref } from 'vue';
import {
  NButton,
  NFlex,
  NInput,
  NModal,
  NPagination,
  NStatistic,
  NText,
  useDialog,
  useMessage,
  NAlert,
} from 'naive-ui';
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';
import { useJsData } from '@/features/js/useJsData';

const message = useMessage();
const dialog = useDialog();

const selectedPlugin = ref<string>('');

type JsDataSearchFormValues = {
  plugin: string | null;
  keyword: string;
};

const defaultJsDataSearchFormValues = (): JsDataSearchFormValues => ({
  plugin: null,
  keyword: '',
});

const dataPage = ref({ page: 1, pageSize: 20 });
const dataKeyword = ref('');
const {
  pluginOptions,
  dataListQuery,
  dataInfoQuery,
  setMutation,
  deleteMutation,
  shrinkMutation,
  invalidateData,
} = useJsData({
  selectedPlugin,
  dataPage,
  dataKeyword,
});

const searchForm = createProSearchForm<JsDataSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultJsDataSearchFormValues()),
  onValueChange: ({ path, value }) => {
    if (path !== 'plugin') return;
    selectedPlugin.value = typeof value === 'string' ? value : '';
    dataPage.value = { ...dataPage.value, page: 1 };
  },
  onSubmit: () => {
    const values = searchForm.values.value;
    selectedPlugin.value = values.plugin ?? '';
    dataKeyword.value = values.keyword.trim();
    dataPage.value = { ...dataPage.value, page: 1 };
    invalidateData();
  },
  onReset: () => {
    selectedPlugin.value = '';
    dataKeyword.value = '';
    dataPage.value = { ...dataPage.value, page: 1 };
  },
});

const searchColumns = computed<ProSearchFormColumns<JsDataSearchFormValues>>(() => [
  {
    label: '插件',
    path: 'plugin',
    field: 'select',
    fieldProps: {
      options: pluginOptions.value,
      placeholder: '选择插件',
      clearable: true,
    },
  },
  {
    label: 'Key',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索 Key（支持 * ? 通配符）',
    },
  },
]);

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

function confirmSave() {
  if (editIsJson.value && jsonError.value) {
    dialog.warning({
      title: 'JSON 格式错误',
      content: '当前数据不是合法 JSON，确定要继续保存吗？',
      positiveText: '仍然保存',
      negativeText: '取消',
      onPositiveClick: saveKey,
    });
    return;
  }
  saveKey();
}

function handleDeleteKey(key: string) {
  dialog.warning({
    title: '删除 Key',
    content: `确认删除 "${key}"？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => deleteKeys([key]),
  });
}

async function saveKey() {
  try {
    await setMutation.mutateAsync({ key: editKey.value, value: editValue.value });
    message.success('已保存');
    showEditModal.value = false;
  } catch {
    message.error('保存失败');
  }
}

async function deleteKeys(keys: string[]) {
  try {
    await deleteMutation.mutateAsync(keys);
    message.success('已删除');
  } catch {
    message.error('删除失败');
  }
}

async function shrinkData() {
  try {
    await shrinkMutation.mutateAsync();
    message.success('数据库已压缩');
  } catch {
    message.error('压缩失败');
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
</script>

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
  width: min(100%, 15rem);
}
.w-80 {
  width: min(100%, 20rem);
}
.mb-4 {
  margin-bottom: 1rem;
}

@media screen and (max-width: 639.9px) {
  .w-60,
  .w-80 {
    width: 100%;
  }
}
</style>
