<template>
  <main class="resource-page">
    <PageHeader
      title="资源管理"
      description="上传图片后可直接复制海豹码，在回复、牌堆或指令中引用本地图片资源。"
    />

    <n-alert v-if="listErrorText" type="error" :bordered="false">
      {{ listErrorText }}
    </n-alert>

    <ResourceListPanel
      class="resource-page__panel"
      :items="items"
      :total="total"
      :loading="resourceListQuery.isFetching.value"
      :query="listQuery"
      :upload-resource="uploadResource"
      :deleting-path="deletingPath"
      :downloading-path="downloadingPath"
      :disabled="isTestMode"
      @update-query="updateListQuery"
      @copy="copySealCode"
      @download="downloadResource"
      @delete="confirmDelete"
      @detail="showDetail"
      @refresh="refreshList"
    />

    <n-drawer v-model:show="detailVisible" :width="detailDrawerWidth" placement="right">
      <n-drawer-content title="资源详情" closable>
        <div v-if="currentResource" class="resource-page__detail">
          <ResourcePreview :item="currentResource" :thumbnail="false" size="large" />
          <n-descriptions :column="1" label-placement="left" bordered size="small">
            <n-descriptions-item label="文件名">
              {{ currentResource.name }}
            </n-descriptions-item>
            <n-descriptions-item label="路径">
              <n-text code>{{ currentResource.path }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="大小">
              {{ formatFileSize(currentResource.size) }}
            </n-descriptions-item>
          </n-descriptions>
          <n-flex size="small" justify="end" wrap>
            <n-button secondary type="info" @click="copySealCode(currentResource)">
              复制海豹码
            </n-button>
            <n-button
              secondary
              type="success"
              :loading="downloadingPath === currentResource.path"
              @click="downloadResource(currentResource)"
            >
              下载
            </n-button>
            <n-button
              secondary
              type="error"
              :loading="deletingPath === currentResource.path"
              @click="confirmDelete(currentResource)"
            >
              删除
            </n-button>
          </n-flex>
        </div>
      </n-drawer-content>
    </n-drawer>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { filesize } from 'filesize';
import { useDialog, useMessage } from 'naive-ui';
import {
  getSdApiV2ResourceDownload,
  getSdApiV2ResourceList,
  postSdApiV2ResourceDelete,
  postSdApiV2ResourceUpload,
  type ResourceItem,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import ResourceListPanel from '@/components/resource/ResourceListPanel.vue';
import ResourcePreview from '@/components/resource/ResourcePreview.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { copyText } from '@/features/clipboard';
import { useResponsiveOverlayWidth } from '@/features/responsive/useResponsiveOverlayWidth';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
  useTestMode,
} from '@/features/testMode/state';
import {
  buildResourceListQuery,
  buildSealImageCode,
  createDefaultResourceListQuery,
  isResourceDetailAvailable,
} from '@/features/resource/viewModel';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();
const { isTestMode } = useTestMode();
const { width: detailDrawerWidth } = useResponsiveOverlayWidth({ maxWidth: 420, gutter: 16 });

const listQuery = reactive(createDefaultResourceListQuery());
const deletingPath = ref('');
const downloadingPath = ref('');
const detailVisible = ref(false);
const currentResource = ref<ResourceItem | null>(null);

const listParams = computed(() => buildResourceListQuery(listQuery));

const resourceListQuery = useQuery({
  queryKey: computed(() => ['resource-list', listParams.value]),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2ResourceList({
      query: listParams.value,
      throwOnError: true,
    });
    return data.item;
  },
});

const items = computed(() => resourceListQuery.data.value?.list ?? []);
const total = computed(() => Number(resourceListQuery.data.value?.total ?? 0));
const formatFileSize = filesize;
const listErrorText = computed(() =>
  resourceListQuery.error.value
    ? getErrorMessage(resourceListQuery.error.value, '加载资源列表失败')
    : ''
);

const invalidateResourceList = () =>
  queryClient.invalidateQueries({
    queryKey: ['resource-list'],
  });

const uploadMutation = useMutation({
  mutationFn: async (file: File) => {
    const { data } = await postSdApiV2ResourceUpload({
      body: {
        files: [file],
      },
      throwOnError: true,
    });
    return data.item;
  },
  onSuccess: async item => {
    if (isTestModeResponse(item)) {
      message.warning('展示模式无法上传资源');
      return;
    }
    if (!item.success) {
      message.error('上传失败');
      return;
    }
    message.success('图片已上传');
    await invalidateResourceList();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, '上传图片失败'));
  },
});

const deleteMutation = useMutation({
  mutationFn: async (item: ResourceItem) => {
    deletingPath.value = item.path;
    const { data } = await postSdApiV2ResourceDelete({
      body: {
        path: item.path,
      },
      throwOnError: true,
    });
    return {
      result: data.item,
      resource: item,
    };
  },
  onSuccess: async ({ result, resource }) => {
    if (!result.success) {
      message.error('删除失败');
      return;
    }
    message.success('资源已删除');
    if (currentResource.value?.path === resource.path) {
      detailVisible.value = false;
      currentResource.value = null;
    }
    await invalidateResourceList();
  },
  onError: error => {
    message.error(getErrorMessage(error, '删除资源失败'));
  },
  onSettled: () => {
    deletingPath.value = '';
  },
});

watch(
  () =>
    [items.value.length, total.value, listQuery.page, resourceListQuery.isFetching.value] as const,
  ([count, itemTotal, page, fetching]) => {
    if (fetching || itemTotal <= 0 || count > 0 || page <= 1) return;
    listQuery.page = page - 1;
  }
);

function updateListQuery(patch: Partial<typeof listQuery>) {
  Object.assign(listQuery, patch);
}

async function uploadResource(file: File) {
  await uploadMutation.mutateAsync(file);
}

function confirmDelete(item: ResourceItem) {
  dialog.warning({
    title: '删除资源',
    content: `确认删除「${item.name}（${item.path}）」吗？删除后无法找回。`,
    positiveText: '删除',
    negativeText: '取消',
    closable: false,
    onPositiveClick: async () => {
      await deleteMutation.mutateAsync(item);
    },
  });
}

async function downloadResource(item: ResourceItem) {
  downloadingPath.value = item.path;
  try {
    await downloadApiFile(
      getSdApiV2ResourceDownload({
        query: {
          path: item.path,
        },
        responseType: 'blob',
        throwOnError: true,
      }),
      item.name
    );
  } catch (error) {
    message.error(getErrorMessage(error, '下载资源失败'));
  } finally {
    downloadingPath.value = '';
  }
}

async function copySealCode(item: ResourceItem) {
  try {
    await copyText(buildSealImageCode(item.path));
    message.success('已复制海豹码');
  } catch {
    message.error('复制失败，请检查浏览器剪贴板权限');
  }
}

function refreshList() {
  void resourceListQuery.refetch();
}

function showDetail(item: ResourceItem) {
  if (!isResourceDetailAvailable(item)) return;
  currentResource.value = item;
  detailVisible.value = true;
}
</script>

<style scoped>
.resource-page {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.resource-page__panel {
  min-width: 0;
}

.resource-page__detail {
  display: grid;
  gap: 16px;
  justify-items: center;
}
</style>
