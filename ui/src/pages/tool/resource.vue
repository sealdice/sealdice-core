<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
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
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import {
  buildResourceListQuery,
  buildSealImageCode,
  createDefaultResourceListQuery,
} from '@/features/resource/viewModel';

const message = useMessage();
const dialog = useDialog();
const queryClient = useQueryClient();

const listQuery = reactive(createDefaultResourceListQuery());
const deletingPath = ref('');
const downloadingPath = ref('');

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
const currentCount = computed(() => items.value.length);
const listErrorText = computed(() => (
  resourceListQuery.error.value ? getErrorMessage(resourceListQuery.error.value, '加载资源列表失败') : ''
));

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
    if (item.testMode) {
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
    return data.item;
  },
  onSuccess: async item => {
    if (!item.success) {
      message.error('删除失败');
      return;
    }
    message.success('资源已删除');
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
  () => [items.value.length, total.value, listQuery.page, resourceListQuery.isFetching.value] as const,
  ([count, itemTotal, page, fetching]) => {
    if (fetching || itemTotal <= 0 || count > 0 || page <= 1) return;
    listQuery.page = page - 1;
  },
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
        parseAs: 'blob',
        throwOnError: true,
      }),
      item.name,
    );
  } catch (error) {
    message.error(getErrorMessage(error, '下载资源失败'));
  } finally {
    downloadingPath.value = '';
  }
}

async function copySealCode(item: ResourceItem) {
  try {
    await navigator.clipboard.writeText(buildSealImageCode(item.path));
    message.success('已复制海豹码');
  } catch {
    message.error('复制失败，请检查浏览器剪贴板权限');
  }
}

function refreshList() {
  void resourceListQuery.refetch();
}
</script>

<template>
  <main class="resource-page">
    <section class="resource-page__hero">
      <div class="resource-page__hero-main">
        <n-tag :bordered="false" type="info">V2 API</n-tag>
        <h1>资源管理</h1>
        <p>上传图片后可直接复制海豹码，在回复、牌堆或指令中引用本地图片资源。</p>
      </div>
      <div class="resource-page__hero-stats">
        <n-statistic label="图片资源" :value="total" />
        <n-statistic label="本页数量" :value="currentCount" />
      </div>
    </section>

    <n-alert v-if="listErrorText" type="error" :bordered="false">
      {{ listErrorText }}
    </n-alert>

    <n-card :bordered="false" class="resource-page__card">
      <ResourceListPanel
        :items="items"
        :total="total"
        :loading="resourceListQuery.isFetching.value"
        :query="listQuery"
        :upload-pending="uploadMutation.isPending.value"
        :deleting-path="deletingPath"
        :downloading-path="downloadingPath"
        @update-query="updateListQuery"
        @upload="uploadResource"
        @copy="copySealCode"
        @download="downloadResource"
        @delete="confirmDelete"
        @refresh="refreshList"
      />
    </n-card>
  </main>
</template>

<style scoped>
.resource-page {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.resource-page__hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  align-items: end;
  padding: 22px;
  overflow: hidden;
  border: 1px solid var(--sd-border-soft);
  border-radius: 22px;
  background:
    radial-gradient(circle at top right, color-mix(in srgb, var(--sd-primary), transparent 78%), transparent 34%),
    linear-gradient(135deg, var(--sd-bg-elevated), var(--sd-bg-elevated-soft));
}

.resource-page__hero-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.resource-page__hero-main h1 {
  margin: 0;
  font-size: clamp(24px, 4vw, 38px);
  line-height: 1.1;
}

.resource-page__hero-main p {
  max-width: 760px;
  margin: 0;
  color: var(--sd-text-secondary);
}

.resource-page__hero-stats {
  display: grid;
  min-width: 220px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--sd-border-soft);
  border-radius: 18px;
  background: var(--sd-bg-elevated-tint);
}

.resource-page__card {
  min-width: 0;
}

@media (max-width: 760px) {
  .resource-page__hero {
    grid-template-columns: 1fr;
    padding: 18px;
  }

  .resource-page__hero-stats {
    min-width: 0;
  }
}
</style>
