<template>
  <main class="package-page">
    <PageHeader title="扩展包管理">
      <n-space>
        <n-button secondary @click="refreshAll" :loading="refreshing">刷新</n-button>
        <n-dropdown :options="reloadMenuOptions" @select="handleReloadSelect">
          <n-button type="primary" :loading="reloading">重载</n-button>
        </n-dropdown>
      </n-space>
    </PageHeader>

    <n-alert v-if="loadErrorText" type="error" class="mb-4">
      {{ loadErrorText }}
    </n-alert>

    <n-tabs :value="activeTab" type="line" animated @update:value="handleTabUpdate">
      <n-tab-pane name="installed" tab="已安装">
        <n-space vertical size="large">
          <ListActions>
            <n-input
              v-model:value="installedKeyword"
              clearable
              placeholder="搜索名称 / ID / 关键词"
              style="width: min(100%, 280px)"
            />
            <n-select
              v-model:value="installedContent"
              :options="contentOptions"
              style="width: min(100%, 160px)"
            />
            <template #end>
              <n-button secondary @click="refreshPackages" :loading="packagesLoading">
                刷新磁盘
              </n-button>
            </template>
          </ListActions>

          <PackageInstalledDataView
            :rows="filteredInstalledPackages"
            :loading="packagesLoading"
            @detail="openDetail"
            @toggle="changePackageState"
            @reload="reloadPackage"
            @uninstall="uninstallPackage"
          />
        </n-space>
      </n-tab-pane>

      <n-tab-pane name="store" tab="商店">
        <n-space vertical size="large">
          <n-space class="package-filter-row" wrap>
            <n-input
              v-model:value="storeKeyword"
              clearable
              placeholder="搜索扩展包名称"
              style="width: min(100%, 280px)"
              @keyup.enter="fetchStorePage"
            />
            <n-button secondary @click="fetchStorePage" :loading="storeLoading">搜索</n-button>
          </n-space>

          <PackageStoreDataView
            :rows="storePackages"
            :loading="storeLoading"
            @preview-install="previewStoreInstall"
          />

          <n-pagination
            v-if="storeTotal > 0"
            v-model:page="storePage"
            v-model:page-size="storePageSize"
            :item-count="storeTotal"
            :page-sizes="[10, 20, 50]"
            show-size-picker
            @update:page="fetchStorePage"
            @update:page-size="fetchStorePage"
          />
        </n-space>
      </n-tab-pane>

      <n-tab-pane name="manage" tab="后端与安装">
        <n-space vertical size="large">
          <n-card title="仓库后端" size="small">
            <n-space vertical>
              <n-space wrap>
                <n-input
                  v-model:value="backendUrlInput"
                  clearable
                  placeholder="输入仓库 URL"
                  style="width: min(100%, 420px)"
                  @keyup.enter="addBackend"
                />
                <n-button type="primary" :loading="backendMutationPending" @click="addBackend"
                  >添加</n-button
                >
              </n-space>
              <n-list bordered>
                <n-list-item v-for="backend in backends" :key="backend.url">
                  <n-space justify="space-between" align="center" style="width: 100%">
                    <div>
                      <n-space align="center">
                        <strong>{{ backend.name || backend.id || backend.url }}</strong>
                        <n-tag size="small" :type="backend.enabled ? 'success' : 'default'">
                          {{ backend.enabled ? '启用' : '禁用' }}
                        </n-tag>
                        <n-tag v-if="backend.official" size="small">官方</n-tag>
                      </n-space>
                      <n-text depth="3">{{ backend.url }}</n-text>
                    </div>
                    <n-space>
                      <n-button
                        secondary
                        size="small"
                        @click="toggleBackend(backend, !backend.enabled)"
                      >
                        {{ backend.enabled ? '禁用' : '启用' }}
                      </n-button>
                      <n-button
                        v-if="!backend.builtin"
                        size="small"
                        tertiary
                        type="error"
                        @click="removeBackend(backend)"
                        >删除</n-button
                      >
                    </n-space>
                  </n-space>
                </n-list-item>
              </n-list>
            </n-space>
          </n-card>

          <n-grid cols="1 m:2" responsive="screen" :x-gap="16" :y-gap="16">
            <n-grid-item>
              <n-card title="上传安装" size="small">
                <n-space vertical>
                  <input
                    ref="uploadInputRef"
                    type="file"
                    accept=".sealpack"
                    hidden
                    @change="handleUploadInput"
                  />
                  <n-space align="center">
                    <n-button secondary @click="uploadInputRef?.click()">选择文件</n-button>
                    <n-text depth="3">{{ uploadFileName || '未选择文件' }}</n-text>
                  </n-space>
                  <n-space>
                    <n-button
                      type="primary"
                      :disabled="!selectedUploadFile"
                      :loading="uploadPreviewLoading"
                      @click="previewUpload"
                      >预览并安装</n-button
                    >
                  </n-space>
                </n-space>
              </n-card>
            </n-grid-item>
            <n-grid-item>
              <n-card title="URL 安装" size="small">
                <n-space vertical>
                  <n-input
                    v-model:value="installUrlInput"
                    clearable
                    placeholder="https://example.com/demo.sealpack"
                  />
                  <n-button
                    type="primary"
                    :disabled="!installUrlInput.trim()"
                    :loading="installUrlLoading"
                    @click="previewUrl"
                    >预览并安装</n-button
                  >
                </n-space>
              </n-card>
            </n-grid-item>
          </n-grid>
        </n-space>
      </n-tab-pane>
    </n-tabs>

    <PackageDetailDrawer
      v-model:show="detailVisible"
      :pkg="currentPackage"
      :config="currentPackageConfig"
      :schema="currentPackageSchema"
      :loading="detailLoading"
      :saving="detailSaving"
      @save-config="savePackageConfig"
    />

    <n-modal
      v-model:show="previewVisible"
      preset="card"
      class="the-dialog"
      :title="previewTitle"
      :mask-closable="!previewBusy"
    >
      <n-spin :show="previewBusy">
        <n-space vertical size="large" v-if="previewData">
          <n-descriptions bordered label-placement="left" :column="2">
            <n-descriptions-item label="包 ID">{{
              previewData.manifest.package.id
            }}</n-descriptions-item>
            <n-descriptions-item label="版本">{{
              previewData.manifest.package.version
            }}</n-descriptions-item>
            <n-descriptions-item label="名称">{{
              previewData.manifest.package.name
            }}</n-descriptions-item>
            <n-descriptions-item label="动作">{{ previewData.installAction }}</n-descriptions-item>
            <n-descriptions-item label="文件数量">{{ previewData.fileCount }}</n-descriptions-item>
            <n-descriptions-item label="安装前版本">{{
              previewData.existingVersion || '-'
            }}</n-descriptions-item>
          </n-descriptions>
          <PackageFileTree :files="previewData.files" />
        </n-space>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button @click="previewVisible = false">关闭</n-button>
          <n-button
            v-if="previewData"
            type="primary"
            :loading="installPreviewConfirmLoading"
            @click="confirmPreviewInstall"
          >
            {{ previewData.installAction === 'upgrade' ? '升级安装' : '安装' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useDialog, useMessage } from 'naive-ui';
import { useQuery } from '@tanstack/vue-query';
import { useRoute, useRouter } from 'vue-router';
import {
  deleteSdApiV2ExtensionStoreBackends,
  getSdApiV2ExtensionConfig,
  getSdApiV2ExtensionConfigSchema,
  getSdApiV2ExtensionPackages,
  getSdApiV2ExtensionStoreBackends,
  getSdApiV2ExtensionStorePage,
  postSdApiV2ExtensionDisable,
  postSdApiV2ExtensionEnable,
  postSdApiV2ExtensionInstallUpload,
  postSdApiV2ExtensionInstallUrl,
  postSdApiV2ExtensionPackagesRefresh,
  postSdApiV2ExtensionPreviewUpload,
  postSdApiV2ExtensionPreviewUrl,
  postSdApiV2ExtensionReload,
  postSdApiV2ExtensionReloadAll,
  postSdApiV2ExtensionReloadContent,
  postSdApiV2ExtensionStoreBackends,
  postSdApiV2ExtensionStoreBackendsDisable,
  postSdApiV2ExtensionStoreBackendsEnable,
  postSdApiV2ExtensionStoreDownload,
  postSdApiV2ExtensionStorePreviewDownload,
  postSdApiV2ExtensionUninstall,
  putSdApiV2ExtensionConfig,
  type Instance,
  type Manifest,
  type PackageUploadPreview,
  type StoreBackend,
  type StorePackage,
} from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { getErrorMessage } from '@/features/auth/error';
import { isTestModeApiError, getTestModeBlockMessage } from '@/features/testMode/state';
import PackageDetailDrawer from '@/components/package/PackageDetailDrawer.vue';
import PackageFileTree from '@/components/package/PackageFileTree.vue';
import PackageInstalledDataView from '@/components/package/PackageInstalledDataView.vue';
import PackageStoreDataView from '@/components/package/PackageStoreDataView.vue';
import ListActions from '@/components/shared/ListActions.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { resolvePackageManagerTab } from '@/features/package/navigation';

type ConfigFieldSchema = {
  type?: string;
  title?: string;
  description?: string;
  default?: unknown;
  secret?: boolean;
  enum?: unknown[] | null;
};

const message = useMessage();
const dialog = useDialog();
const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const activeTab = computed(() => resolvePackageManagerTab(route.query.tab));

function handleTabUpdate(value: string | number) {
  const tab = resolvePackageManagerTab(value);
  const query = { ...route.query };
  if (tab === 'installed') delete query.tab;
  else query.tab = tab;
  void router.replace({ query });
}

const packagesLoading = ref(false);
const storeLoading = ref(false);
const refreshing = ref(false);
const reloading = ref(false);
const backendMutationPending = ref(false);
const detailLoading = ref(false);
const detailSaving = ref(false);
const uploadPreviewLoading = ref(false);
const uploadInstallLoading = ref(false);
const installUrlLoading = ref(false);
const previewBusy = ref(false);
const installPreviewConfirmLoading = ref(false);

const installedKeyword = ref('');
const installedContent = ref<'all' | 'scripts' | 'decks' | 'reply' | 'helpdoc' | 'templates'>(
  'all'
);
const storeKeyword = ref('');
const storePage = ref(1);
const storePageSize = ref(20);
const backendUrlInput = ref('');
const installUrlInput = ref('');
const selectedUploadFile = ref<File | null>(null);
const uploadFileName = ref('');
const uploadInputRef = ref<HTMLInputElement | null>(null);

const detailVisible = ref(false);
const currentPackage = ref<Instance | null>(null);
const currentPackageConfig = ref<Record<string, unknown> | null>(null);
const currentPackageSchema = ref<Record<string, ConfigFieldSchema> | null>(null);

const previewVisible = ref(false);
const previewTitle = ref('安装预览');
const previewData = ref<PackageUploadPreview | null>(null);
const previewSource = ref<'store' | 'upload' | 'url' | 'none'>('none');
const previewStoreTarget = ref<StorePackage | null>(null);
const previewUrlTarget = ref('');

const packagesQuery = useQuery({
  queryKey: ['extension-packages'],
  queryFn: async () => {
    const { data } = await getSdApiV2ExtensionPackages({ throwOnError: true });
    return data.item.items ?? [];
  },
  enabled: authStore.hasAccessToken,
});

const storeBackendsQuery = useQuery({
  queryKey: ['extension-store-backends'],
  queryFn: async () => {
    const { data } = await getSdApiV2ExtensionStoreBackends({ throwOnError: true });
    return data.item.items ?? [];
  },
  enabled: authStore.hasAccessToken,
});

const storePageQuery = useQuery({
  queryKey: computed(() => [
    'extension-store-page',
    storeKeyword.value,
    storePage.value,
    storePageSize.value,
  ]),
  queryFn: async () => {
    const { data } = await getSdApiV2ExtensionStorePage({
      query: {
        name: storeKeyword.value,
        pageNum: storePage.value,
        pageSize: storePageSize.value,
      },
      throwOnError: true,
    });
    return data.item;
  },
  enabled: authStore.hasAccessToken,
});

const packages = computed(() => packagesQuery.data.value ?? []);
const backends = computed(() => storeBackendsQuery.data.value ?? []);
const storePackages = computed(() => storePageQuery.data.value?.items ?? []);
const storeTotal = computed(() =>
  storePageQuery.data.value?.next
    ? storePage.value * storePageSize.value + 1
    : storePackages.value.length
);

const filteredInstalledPackages = computed(() =>
  packages.value.filter(pkg => {
    if (!matchesContentFilter(pkg.manifest, installedContent.value)) return false;
    const keyword = installedKeyword.value.trim().toLowerCase();
    if (!keyword) return true;
    const haystack = [
      pkg.manifest.package.name,
      pkg.manifest.package.id,
      pkg.manifest.package.version,
      ...(pkg.manifest.package.authors ?? []),
      ...(pkg.manifest.package.keywords ?? []),
    ]
      .join(' ')
      .toLowerCase();
    return haystack.includes(keyword);
  })
);

const loadErrorText = computed(() => {
  if (packagesQuery.isError.value)
    return getErrorMessage(packagesQuery.error.value, '扩展包读取失败');
  if (storeBackendsQuery.isError.value)
    return getErrorMessage(storeBackendsQuery.error.value, '仓库后端读取失败');
  if (storePageQuery.isError.value)
    return getErrorMessage(storePageQuery.error.value, '商店列表读取失败');
  return '';
});

const contentOptions = [
  { label: '全部内容', value: 'all' },
  { label: '脚本', value: 'scripts' },
  { label: '牌堆', value: 'decks' },
  { label: '自定义回复', value: 'reply' },
  { label: '帮助文档', value: 'helpdoc' },
  { label: '模板', value: 'templates' },
];

const reloadMenuOptions = [
  { label: '重载全部', key: 'all' },
  { label: '重载脚本', key: 'scripts' },
  { label: '重载牌堆', key: 'decks' },
  { label: '重载自定义回复', key: 'reply' },
  { label: '重载帮助文档', key: 'helpdoc' },
  { label: '重载模板', key: 'templates' },
];

watch(
  () => authStore.hasAccessToken,
  hasToken => {
    if (!hasToken) {
      previewVisible.value = false;
      detailVisible.value = false;
    }
  },
  { immediate: true }
);

onMounted(async () => {
  await refreshAll();
});

async function refreshAll() {
  refreshing.value = true;
  try {
    await Promise.all([
      packagesQuery.refetch(),
      storeBackendsQuery.refetch(),
      storePageQuery.refetch(),
    ]);
  } finally {
    refreshing.value = false;
  }
}

async function refreshPackages() {
  packagesLoading.value = true;
  try {
    await postSdApiV2ExtensionPackagesRefresh({ throwOnError: true });
    await packagesQuery.refetch();
  } catch (error) {
    handleError(error, '刷新扩展包失败');
  } finally {
    packagesLoading.value = false;
  }
}

async function fetchStorePage() {
  storeLoading.value = true;
  try {
    await storePageQuery.refetch();
  } finally {
    storeLoading.value = false;
  }
}

async function openDetail(pkg: Instance) {
  detailVisible.value = true;
  currentPackage.value = pkg;
  detailLoading.value = true;
  currentPackageConfig.value = null;
  currentPackageSchema.value = null;
  try {
    const [configResp, schemaResp] = await Promise.all([
      getSdApiV2ExtensionConfig({ query: { id: pkg.manifest.package.id }, throwOnError: true }),
      getSdApiV2ExtensionConfigSchema({
        query: { id: pkg.manifest.package.id },
        throwOnError: true,
      }),
    ]);
    currentPackageConfig.value = configResp.data.item;
    currentPackageSchema.value = schemaResp.data.item;
  } catch (error) {
    handleError(error, '读取扩展包详情失败');
    detailVisible.value = false;
  } finally {
    detailLoading.value = false;
  }
}

async function savePackageConfig(nextConfig: Record<string, unknown>) {
  if (!currentPackage.value) return;
  detailSaving.value = true;
  try {
    await putSdApiV2ExtensionConfig({
      query: { id: currentPackage.value.manifest.package.id },
      body: nextConfig,
      throwOnError: true,
    });
    message.success('配置已保存');
    await packagesQuery.refetch();
    currentPackageConfig.value = nextConfig;
  } catch (error) {
    handleError(error, '保存配置失败');
  } finally {
    detailSaving.value = false;
  }
}

async function changePackageState(pkg: Instance, enable: boolean) {
  try {
    if (enable) {
      await postSdApiV2ExtensionEnable({
        body: { id: pkg.manifest.package.id },
        throwOnError: true,
      });
    } else {
      await postSdApiV2ExtensionDisable({
        body: { id: pkg.manifest.package.id },
        throwOnError: true,
      });
    }
    message.success('状态已更新');
    await packagesQuery.refetch();
  } catch (error) {
    handleError(error, enable ? '启用失败' : '禁用失败');
  }
}

async function reloadPackage(pkg: Instance) {
  try {
    await postSdApiV2ExtensionReload({ body: { id: pkg.manifest.package.id }, throwOnError: true });
    message.success('已重载');
    await packagesQuery.refetch();
  } catch (error) {
    handleError(error, '重载失败');
  }
}

async function uninstallPackage(pkg: Instance) {
  dialog.warning({
    title: '卸载扩展包',
    content: `确认卸载 ${pkg.manifest.package.name} 吗？`,
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await postSdApiV2ExtensionUninstall({
          body: { id: pkg.manifest.package.id, mode: 'full' },
          throwOnError: true,
        });
        message.success('已卸载');
        await packagesQuery.refetch();
      } catch (error) {
        handleError(error, '卸载失败');
      }
    },
  });
}

async function handleReloadSelect(key: string) {
  try {
    reloading.value = true;
    if (key === 'all') {
      await postSdApiV2ExtensionReloadAll({ throwOnError: true });
    } else {
      await postSdApiV2ExtensionReloadContent({ body: { content: key }, throwOnError: true });
    }
    message.success('已重载');
    await packagesQuery.refetch();
  } catch (error) {
    handleError(error, '重载失败');
  } finally {
    reloading.value = false;
  }
}

async function addBackend() {
  if (!backendUrlInput.value.trim()) return;
  backendMutationPending.value = true;
  try {
    await postSdApiV2ExtensionStoreBackends({
      body: { url: backendUrlInput.value.trim() },
      throwOnError: true,
    });
    backendUrlInput.value = '';
    message.success('后端已添加');
    await storeBackendsQuery.refetch();
  } catch (error) {
    handleError(error, '添加后端失败');
  } finally {
    backendMutationPending.value = false;
  }
}

async function toggleBackend(backend: StoreBackend, enabled: boolean) {
  backendMutationPending.value = true;
  try {
    const body = { id: backend.id || '', backendID: backend.id || '', url: backend.url || '' };
    if (enabled) {
      await postSdApiV2ExtensionStoreBackendsEnable({ body, throwOnError: true });
    } else {
      await postSdApiV2ExtensionStoreBackendsDisable({ body, throwOnError: true });
    }
    await storeBackendsQuery.refetch();
  } catch (error) {
    handleError(error, '修改后端失败');
  } finally {
    backendMutationPending.value = false;
  }
}

async function removeBackend(backend: StoreBackend) {
  backendMutationPending.value = true;
  try {
    await deleteSdApiV2ExtensionStoreBackends({
      body: { id: backend.id || '', backendID: backend.id || '', url: backend.url || '' },
      throwOnError: true,
    });
    await storeBackendsQuery.refetch();
  } catch (error) {
    handleError(error, '删除后端失败');
  } finally {
    backendMutationPending.value = false;
  }
}

function handleUploadInput(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] ?? null;
  selectedUploadFile.value = file;
  uploadFileName.value = file?.name ?? '';
}

async function previewUpload() {
  if (!selectedUploadFile.value) return;
  uploadPreviewLoading.value = true;
  try {
    const { data } = await postSdApiV2ExtensionPreviewUpload({
      body: selectedUploadFile.value,
      throwOnError: true,
    });
    previewSource.value = 'upload';
    previewTitle.value = '上传预览';
    previewData.value = data.item;
    previewVisible.value = true;
  } catch (error) {
    handleError(error, '预览失败');
  } finally {
    uploadPreviewLoading.value = false;
  }
}

async function installUpload(): Promise<boolean> {
  if (!selectedUploadFile.value) return false;
  uploadInstallLoading.value = true;
  try {
    await postSdApiV2ExtensionInstallUpload({
      body: selectedUploadFile.value,
      throwOnError: true,
    });
    message.success('已安装');
    await packagesQuery.refetch();
    return true;
  } catch (error) {
    handleError(error, '上传安装失败');
    return false;
  } finally {
    uploadInstallLoading.value = false;
  }
}

async function installByUrl(url = installUrlInput.value.trim()): Promise<boolean> {
  if (!url) return false;
  installUrlLoading.value = true;
  try {
    await postSdApiV2ExtensionInstallUrl({
      body: { url },
      throwOnError: true,
    });
    message.success('已安装');
    installUrlInput.value = '';
    await packagesQuery.refetch();
    return true;
  } catch (error) {
    handleError(error, 'URL 安装失败');
    return false;
  } finally {
    installUrlLoading.value = false;
  }
}

async function previewUrl() {
  const url = installUrlInput.value.trim();
  if (!url) return;
  installUrlLoading.value = true;
  try {
    const { data } = await postSdApiV2ExtensionPreviewUrl({
      body: { url },
      throwOnError: true,
    });
    previewSource.value = 'url';
    previewUrlTarget.value = url;
    previewTitle.value = 'URL 安装预览';
    previewData.value = data.item;
    previewVisible.value = true;
  } catch (error) {
    handleError(error, 'URL 预览失败');
  } finally {
    installUrlLoading.value = false;
  }
}

async function previewStoreInstall(pkg: StorePackage) {
  previewBusy.value = true;
  previewSource.value = 'store';
  previewStoreTarget.value = pkg;
  previewTitle.value = `${pkg.name} 安装预览`;
  try {
    const { data } = await postSdApiV2ExtensionStorePreviewDownload({
      body: { id: pkg.id, version: pkg.version },
      throwOnError: true,
    });
    previewData.value = data.item;
    previewVisible.value = true;
  } catch (error) {
    handleError(error, '获取商店预览失败');
  } finally {
    previewBusy.value = false;
  }
}

async function installStorePackage(pkg: StorePackage): Promise<boolean> {
  previewStoreTarget.value = pkg;
  try {
    await postSdApiV2ExtensionStoreDownload({
      body: { id: pkg.id, version: pkg.version },
      throwOnError: true,
    });
    message.success('已安装');
    await packagesQuery.refetch();
    return true;
  } catch (error) {
    handleError(error, '商店安装失败');
    return false;
  }
}

async function confirmPreviewInstall() {
  if (!previewData.value) return;
  installPreviewConfirmLoading.value = true;
  try {
    let installed = false;
    if (previewSource.value === 'store' && previewStoreTarget.value) {
      installed = await installStorePackage(previewStoreTarget.value);
    } else if (previewSource.value === 'upload' && selectedUploadFile.value) {
      installed = await installUpload();
    } else if (previewSource.value === 'url' && previewUrlTarget.value) {
      installed = await installByUrl(previewUrlTarget.value);
    }
    if (installed) {
      previewVisible.value = false;
    }
  } finally {
    installPreviewConfirmLoading.value = false;
  }
}

function matchesContentFilter(manifest: Manifest, filter: string) {
  if (filter === 'all') return true;
  const contents = manifest.contents ?? {};
  return (
    Array.isArray((contents as Record<string, unknown>)[filter]) &&
    ((contents as Record<string, string[]>)[filter]?.length ?? 0) > 0
  );
}

function handleError(error: unknown, fallback: string) {
  if (isTestModeApiError(error)) {
    message.warning(getTestModeBlockMessage(error));
    return;
  }
  message.error(getErrorMessage(error, fallback));
}
</script>

<style scoped>
.package-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.package-filter-row {
  max-width: 100%;
}
</style>
