<template>
  <main class="package-page sd-page-flow">
    <PageHeader title="扩展包管理">
      <n-space>
        <n-button secondary @click="refreshAll" :loading="refreshing">刷新</n-button>
        <n-dropdown :options="reloadMenuOptions" @select="handleReloadSelect">
          <n-button
            :type="hasPendingReload ? 'warning' : 'primary'"
            :loading="reloading"
            :aria-label="hasPendingReload ? `重载，${pendingReloadCount} 个扩展包待生效` : '重载'"
          >
            <template #icon>
              <n-icon>
                <i-tabler-alert-triangle v-if="hasPendingReload" />
                <i-tabler-refresh v-else />
              </n-icon>
            </template>
            {{ hasPendingReload ? `重载（${pendingReloadCount}）` : '重载' }}
          </n-button>
        </n-dropdown>
      </n-space>
    </PageHeader>

    <TipBox v-if="loadErrorText" type="error">
      {{ loadErrorText }}
    </TipBox>

    <n-tabs
      :value="activeTab"
      class="sd-scrollable-tabs"
      type="line"
      animated
      @update:value="handleTabUpdate"
    >
      <n-tab-pane name="installed" tab="已安装">
        <ListWorkspace>
          <QueryToolbar
            :form="installedSearchForm"
            :columns="installedSearchColumns"
            :loading="packagesLoading"
            cols="1 s:2 l:3"
          />

          <ListActions>
            <template #end>
              <n-button secondary @click="refreshPackages" :loading="packagesLoading">
                刷新磁盘
              </n-button>
            </template>
          </ListActions>

          <ListPanel>
            <template #toolbar>
              <ResultToolbar>
                <template #meta>
                  <n-text depth="3">共 {{ filteredInstalledPackages.length }} 项</n-text>
                </template>
              </ResultToolbar>
            </template>
            <PackageInstalledDataView
              :rows="filteredInstalledPackages"
              :loading="packagesLoading"
              @detail="openDetail"
              @toggle="changePackageState"
              @reload="reloadPackage"
              @uninstall="uninstallPackage"
            />
          </ListPanel>
        </ListWorkspace>
      </n-tab-pane>

      <n-tab-pane name="store" tab="商店">
        <ListWorkspace>
          <QueryToolbar
            :form="storeSearchForm"
            :columns="storeSearchColumns"
            :loading="storeLoading || storePageQuery.isFetching.value"
            cols="1"
          />

          <div class="store-category-bar">
            <n-text depth="3" class="store-category-label">分类</n-text>
            <n-radio-group
              v-model:value="storeCategory"
              aria-label="商店分类"
              size="small"
              @update:value="handleStoreCategoryChange"
            >
              <n-radio-button
                v-for="option in storeCategoryOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </n-radio-button>
            </n-radio-group>
          </div>

          <ListPanel>
            <PackageStoreDataView
              :rows="visibleStorePackages"
              :loading="storeLoading"
              @detail="openStoreDetail"
              @install="openStoreInstall"
              @uninstall="confirmStoreUninstall"
            />
          </ListPanel>

          <n-pagination
            v-if="showStorePagination"
            class="package-pagination"
            v-model:page="storePage"
            v-model:page-size="storePageSize"
            :item-count="storeTotal"
            :page-sizes="[10, 20, 50]"
            show-size-picker
            @update:page="fetchStorePage"
            @update:page-size="handleStorePageSizeChange"
          />
        </ListWorkspace>
      </n-tab-pane>

      <n-tab-pane name="manage" tab="后端与安装">
        <div class="package-setting-groups">
          <SettingCategoryBox title="仓库后端" padded>
            <RepeatableList add-label="添加仓库后端">
              <RepeatableItem
                v-for="backend in backends"
                :key="backend.url"
                :title="backend.name || backend.id || backend.url"
                :show-enabled="true"
                :enabled="backend.enabled"
                enabled-label="启用仓库后端"
                :removable="!backend.builtin"
                remove-label="删除仓库后端"
                :disabled="backendMutationPending"
                @update:enabled="toggleBackend(backend, $event)"
                @remove="removeBackend(backend)"
              >
                <n-flex align="center" size="small" wrap>
                  <n-text depth="3" class="package-backend-url">{{ backend.url }}</n-text>
                  <n-tag v-if="backend.official" size="small" :bordered="false">官方</n-tag>
                </n-flex>
              </RepeatableItem>

              <template #footer>
                <div class="package-backend-add">
                  <n-input
                    v-model:value="backendUrlInput"
                    clearable
                    placeholder="输入仓库 URL"
                    @keyup.enter="addBackend"
                  />
                  <n-button
                    type="primary"
                    secondary
                    size="small"
                    :loading="backendMutationPending"
                    @click="addBackend"
                  >
                    <template #icon><i-tabler-plus /></template>
                    添加仓库后端
                  </n-button>
                </div>
              </template>
            </RepeatableList>
          </SettingCategoryBox>

          <div class="package-install-groups">
            <SettingCategoryBox title="上传安装" padded>
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
            </SettingCategoryBox>
            <SettingCategoryBox title="URL 安装" padded>
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
            </SettingCategoryBox>
          </div>
        </div>
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
      v-model:show="storeDetailVisible"
      preset="card"
      class="the-dialog"
      title="扩展包详情"
      :mask-closable="true"
    >
      <template v-if="currentStorePackage">
        <n-descriptions bordered label-placement="left" :column="2">
          <n-descriptions-item label="名称">{{ currentStorePackage.name }}</n-descriptions-item>
          <n-descriptions-item label="包 ID">{{ currentStorePackage.id }}</n-descriptions-item>
          <n-descriptions-item label="版本">{{ currentStorePackage.version }}</n-descriptions-item>
          <n-descriptions-item label="作者">{{
            currentStorePackage.authors?.join(' / ') || '-'
          }}</n-descriptions-item>
          <n-descriptions-item label="分类">{{
            currentStorePackage.storeAssets?.category || '-'
          }}</n-descriptions-item>
          <n-descriptions-item label="安装状态">{{
            currentStorePackage.installed ? '已安装' : '未安装'
          }}</n-descriptions-item>
          <n-descriptions-item label="许可证">{{
            currentStorePackage.license || '-'
          }}</n-descriptions-item>
          <n-descriptions-item label="文件大小">{{
            formatStoreFileSize(currentStorePackage.download?.size ?? 0)
          }}</n-descriptions-item>
          <n-descriptions-item label="更新时间">{{
            formatUpdateTime(currentStorePackage)
          }}</n-descriptions-item>
        </n-descriptions>
        <p v-if="currentStorePackage.description" class="package-store-description">
          {{ currentStorePackage.description }}
        </p>
      </template>
      <template #footer>
        <n-space justify="end" wrap>
          <n-button @click="storeDetailVisible = false">关闭</n-button>
          <n-button
            v-if="currentStorePackage"
            type="primary"
            @click="openStoreInstall(currentStorePackage)"
          >
            {{ currentStorePackage.installed ? '重装' : '安装' }}
          </n-button>
          <n-button
            v-if="currentStorePackage?.installed"
            type="error"
            secondary
            @click="confirmStoreUninstall(currentStorePackage)"
          >
            卸载
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="previewVisible"
      preset="card"
      class="the-dialog"
      :title="previewTitle"
      :closable="!installPreviewConfirmLoading"
      :mask-closable="!previewBusy && !installPreviewConfirmLoading"
    >
      <n-spin :show="previewBusy">
        <n-space vertical size="large" v-if="previewData || previewStoreTarget">
          <template v-if="previewData">
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
              <n-descriptions-item label="动作">{{
                previewData.installAction
              }}</n-descriptions-item>
              <n-descriptions-item label="文件数量">{{
                previewData.fileCount
              }}</n-descriptions-item>
              <n-descriptions-item label="安装前版本">{{
                previewData.existingVersion || '-'
              }}</n-descriptions-item>
            </n-descriptions>
            <PackageFileTree :files="previewData.files" />
          </template>

          <template v-else-if="previewStoreTarget">
            <n-descriptions bordered label-placement="left" :column="2">
              <n-descriptions-item label="名称">{{ previewStoreTarget.name }}</n-descriptions-item>
              <n-descriptions-item label="包 ID">{{ previewStoreTarget.id }}</n-descriptions-item>
              <n-descriptions-item label="目标版本">{{
                previewStoreTarget.version
              }}</n-descriptions-item>
              <n-descriptions-item label="当前状态">{{
                previewStoreTarget.installed ? '已安装，将重装' : '未安装'
              }}</n-descriptions-item>
              <n-descriptions-item label="作者">{{
                previewStoreTarget.authors?.join(' / ') || '-'
              }}</n-descriptions-item>
              <n-descriptions-item label="分类">{{
                previewStoreTarget.storeAssets?.category || '-'
              }}</n-descriptions-item>
              <n-descriptions-item label="内容">{{
                previewStoreTarget.contents?.join('、') || '-'
              }}</n-descriptions-item>
              <n-descriptions-item label="文件大小">{{
                formatStoreFileSize(previewStoreTarget.download?.size ?? 0)
              }}</n-descriptions-item>
            </n-descriptions>
            <p v-if="previewStoreTarget.description" class="package-store-description">
              {{ previewStoreTarget.description }}
            </p>
          </template>

          <n-divider />
          <n-space vertical size="small" class="install-options">
            <n-checkbox
              v-model:checked="installAutoEnable"
              :disabled="installPreviewConfirmLoading"
            >
              安装后启用
            </n-checkbox>
            <n-checkbox
              v-model:checked="installAutoReload"
              :disabled="installPreviewConfirmLoading || !installAutoEnable"
            >
              安装并启用后自动重载
            </n-checkbox>
            <n-text depth="3"> 如果扩展包很多，重载可能需要较长时间，特别是帮助文档。 </n-text>
          </n-space>
        </n-space>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button :disabled="installPreviewConfirmLoading" @click="previewVisible = false">
            关闭
          </n-button>
          <n-button
            v-if="previewData || previewStoreTarget"
            type="primary"
            :loading="installPreviewConfirmLoading"
            @click="confirmPreviewInstall"
          >
            {{
              previewData
                ? previewData.installAction === 'upgrade'
                  ? '升级安装'
                  : '安装'
                : previewStoreTarget?.installed
                  ? '重装'
                  : '安装'
            }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { usePackageDetail } from '@/features/package/usePackageDetail';
import { type DropdownDividerOption, type DropdownOption, useDialog, useMessage } from 'naive-ui';
import { usePackageCatalog } from '@/features/package/usePackageCatalog';
import { useRoute, useRouter } from 'vue-router';
import { createProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';
import {
  deleteSdApiV2ExtensionStoreBackends,
  postSdApiV2ExtensionDisable,
  postSdApiV2ExtensionEnable,
  postSdApiV2ExtensionInstallUrl,
  postSdApiV2ExtensionPreviewUpload,
  postSdApiV2ExtensionPreviewUrl,
  postSdApiV2ExtensionReload,
  postSdApiV2ExtensionReloadAll,
  postSdApiV2ExtensionReloadContent,
  postSdApiV2ExtensionStoreBackends,
  postSdApiV2ExtensionStoreBackendsDisable,
  postSdApiV2ExtensionStoreBackendsEnable,
  postSdApiV2ExtensionStoreDownload,
  postSdApiV2ExtensionUninstall,
  type Instance,
  type Manifest,
  type PackageUploadPreview,
  type StoreBackend,
  type StorePackage,
} from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { getErrorMessage } from '@/features/auth/error';
import { isApiErrorHandled } from '@/api/errors';
import { useUploadStore } from '@/features/upload/store';
import {
  installPackage,
  type InstallOptions,
  type InstallResult,
} from '@/features/package/install';
import { isTestModeApiError, getTestModeBlockMessage } from '@/features/testMode/state';
import PackageDetailDrawer from '@/components/package/PackageDetailDrawer.vue';
import PackageFileTree from '@/components/package/PackageFileTree.vue';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import PackageInstalledDataView from '@/components/package/PackageInstalledDataView.vue';
import PackageStoreDataView from '@/components/package/PackageStoreDataView.vue';
import ListActions from '@/components/shared/ListActions.vue';
import ListPanel from '@/components/shared/ListPanel.vue';
import ListWorkspace from '@/components/shared/ListWorkspace.vue';
import {
  getCursorPaginationItemCount,
  shouldShowListPagination,
} from '@/components/shared/listPagination';
import PageHeader from '@/components/shared/PageHeader.vue';
import QueryToolbar from '@/components/shared/QueryToolbar.vue';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
import ResultToolbar from '@/components/shared/ResultToolbar.vue';
import { resolvePackageManagerTab } from '@/features/package/navigation';
import { cloneSearchFormValues } from '@/features/searchForm/viewModel';
import TipBox from '@/components/shared/TipBox.vue';
import { notifyError } from '@/features/feedback/notifyError';

const uploads = useUploadStore();
const message = useMessage();
const dialog = useDialog();
const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const activeTab = computed(() => resolvePackageManagerTab(route.query.tab));

type InstalledSearchFormValues = {
  keyword: string;
  content: 'all' | 'scripts' | 'decks' | 'reply' | 'helpdoc' | 'templates';
};

type StoreSearchFormValues = {
  keyword: string;
};

const defaultInstalledSearchValues = (): InstalledSearchFormValues => ({
  keyword: '',
  content: 'all',
});

const defaultStoreSearchValues = (): StoreSearchFormValues => ({
  keyword: '',
});

const contentOptions = [
  { label: '全部内容', value: 'all' },
  { label: '脚本', value: 'scripts' },
  { label: '牌堆', value: 'decks' },
  { label: '自定义回复', value: 'reply' },
  { label: '帮助文档', value: 'helpdoc' },
  { label: '模板', value: 'templates' },
];

function handleTabUpdate(value: string | number) {
  const tab = resolvePackageManagerTab(value);
  const query = { ...route.query };
  if (tab === 'installed') delete query.tab;
  else query.tab = tab;
  void router.replace({ query });
}

const reloading = ref(false);
const detachedReloadPending = ref(false);
const backendMutationPending = ref(false);
const uploadPreviewLoading = ref(false);
const uploadInstallLoading = ref(false);
const installUrlLoading = ref(false);
const previewBusy = ref(false);
const installPreviewConfirmLoading = ref(false);
const installAutoEnable = ref(true);
const installAutoReload = ref(false);

const installedKeyword = ref('');
const installedContent = ref<'all' | 'scripts' | 'decks' | 'reply' | 'helpdoc' | 'templates'>(
  'all'
);
const storeKeyword = ref('');
const storeCategory = ref('');
const knownStoreCategories = ref(new Set<string>());
const storePage = ref(1);
const storePageSize = ref(20);
const backendUrlInput = ref('');
const installUrlInput = ref('');
const selectedUploadFile = ref<File | null>(null);
const uploadFileName = ref('');
const uploadInputRef = ref<HTMLInputElement | null>(null);

const installedSearchForm = createProSearchForm<InstalledSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultInstalledSearchValues()),
  onSubmit: values => {
    installedKeyword.value = values.keyword;
    installedContent.value = values.content;
  },
  onReset: () => {
    const defaults = defaultInstalledSearchValues();
    installedKeyword.value = defaults.keyword;
    installedContent.value = defaults.content;
  },
});

const installedSearchColumns: ProSearchFormColumns<InstalledSearchFormValues> = [
  {
    label: '关键字',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索名称 / ID / 关键词',
    },
  },
  {
    label: '内容',
    path: 'content',
    field: 'select',
    fieldProps: {
      options: contentOptions,
    },
  },
];

const storeSearchForm = createProSearchForm<StoreSearchFormValues>({
  initialValues: cloneSearchFormValues(defaultStoreSearchValues()),
  onSubmit: async values => {
    await updateStoreSearch(values.keyword);
  },
  onReset: async () => {
    await updateStoreSearch(defaultStoreSearchValues().keyword);
  },
});

const storeSearchColumns: ProSearchFormColumns<StoreSearchFormValues> = [
  {
    label: '名称',
    path: 'keyword',
    field: 'input',
    fieldProps: {
      clearable: true,
      placeholder: '搜索扩展包名称',
    },
  },
];

const storeDetailVisible = ref(false);
const currentStorePackage = ref<StorePackage | null>(null);

const previewVisible = ref(false);
const previewTitle = ref('安装预览');
const previewData = ref<PackageUploadPreview | null>(null);
const previewSource = ref<'store' | 'upload' | 'url' | 'none'>('none');
const previewStoreTarget = ref<StorePackage | null>(null);
const previewUrlTarget = ref('');

const {
  packagesQuery,
  storeBackendsQuery,
  storePageQuery,
  loadErrorText,
  refreshing,
  packagesLoading,
  storeLoading,
  refreshAll,
  fetchStorePage,
  refreshPackages: refreshPackageCatalog,
} = usePackageCatalog(() => ({
  name: storeKeyword.value,
  category: storeCategory.value || undefined,
  pageNum: storePage.value,
  pageSize: storePageSize.value,
}));

const {
  visible: detailVisible,
  pkg: currentPackage,
  config: currentPackageConfig,
  schema: currentPackageSchema,
  loading: detailLoading,
  saving: detailSaving,
  open: openDetail,
  saveConfig: savePackageConfig,
} = usePackageDetail({
  onSaved: async () => {
    message.success('配置已保存');
    await packagesQuery.refetch();
  },
  onError: handleError,
});

const packages = computed(() => packagesQuery.data.value ?? []);
const pendingReloadPackages = computed(() =>
  packages.value.filter(pkg => Boolean(pkg.pendingReload?.length))
);
const hasPendingReload = computed(
  () => pendingReloadPackages.value.length > 0 || detachedReloadPending.value
);
const pendingReloadCount = computed(
  () => pendingReloadPackages.value.length + (detachedReloadPending.value ? 1 : 0)
);
const backends = computed(() => storeBackendsQuery.data.value ?? []);
const storePackages = computed(() => storePageQuery.data.value?.items ?? []);
const storeHasNext = computed(() => Boolean(storePageQuery.data.value?.next));
const storeTotal = computed(() =>
  getCursorPaginationItemCount({
    page: storePage.value,
    pageSize: storePageSize.value,
    itemCount: storePackages.value.length,
    hasNext: storeHasNext.value,
  })
);
const showStorePagination = computed(() =>
  shouldShowListPagination({
    page: storePage.value,
    pageSize: storePageSize.value,
    hasNext: storeHasNext.value,
  })
);

const storeCategoryOptions = computed(() => [
  { label: '全部分类', value: '' },
  ...[...knownStoreCategories.value].sort().map(category => ({ label: category, value: category })),
]);

const visibleStorePackages = computed(() => {
  if (!storeCategory.value) return storePackages.value;
  const category = storeCategory.value;
  return storePackages.value.filter(row => (row.storeAssets?.category ?? '').trim() === category);
});

watch(
  storePackages,
  rows => {
    const next = new Set(knownStoreCategories.value);
    for (const row of rows) {
      const category = row.storeAssets?.category?.trim();
      if (category) next.add(category);
    }
    knownStoreCategories.value = next;
  },
  { immediate: true }
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

type ReloadMenuOption = DropdownOption | DropdownDividerOption;

const reloadMenuOptions = computed<ReloadMenuOption[]>(() => {
  const options: ReloadMenuOption[] = [];
  if (hasPendingReload.value) {
    options.push({
      label: `重载待生效扩展包（${pendingReloadCount.value}）`,
      key: 'pending',
    });
    options.push({ type: 'divider', key: 'pending-divider' });
  }
  options.push(
    { label: '重载全部', key: 'all' },
    { label: '重载脚本', key: 'scripts' },
    { label: '重载牌堆', key: 'decks' },
    { label: '重载自定义回复', key: 'reply' },
    { label: '重载帮助文档', key: 'helpdoc' },
    { label: '重载模板', key: 'templates' }
  );
  return options;
});

watch(
  () => authStore.hasAccessToken,
  hasToken => {
    if (!hasToken) {
      previewVisible.value = false;
      previewStoreTarget.value = null;
      storeDetailVisible.value = false;
      detachedReloadPending.value = false;
    }
  },
  { immediate: true }
);

watch(installAutoEnable, enabled => {
  if (!enabled) installAutoReload.value = false;
});

async function refreshPackages() {
  try {
    await refreshPackageCatalog();
  } catch (error) {
    handleError(error, '刷新扩展包失败');
  }
}

async function updateStoreSearch(keyword: string) {
  const normalizedKeyword = keyword.trim();
  const queryChanged = storeKeyword.value !== normalizedKeyword || storePage.value !== 1;
  storeKeyword.value = normalizedKeyword;
  storePage.value = 1;
  if (!queryChanged) await fetchStorePage();
}

async function handleStoreCategoryChange(category: string) {
  storeCategory.value = category;
  storePage.value = 1;
  await fetchStorePage();
}

async function handleStorePageSizeChange() {
  storePage.value = 1;
  await fetchStorePage();
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

function packageNeedsReload(id: string) {
  const pkg = packages.value.find(item => item.manifest.package.id === id);
  return Boolean(pkg && (pkg.state === 'enabled' || pkg.pendingReload?.length));
}

function notifyUninstallResult(reloadNeeded: boolean, refreshSucceeded: boolean) {
  if (reloadNeeded) detachedReloadPending.value = true;
  const resultText = reloadNeeded ? '已卸载，但运行时资源仍需重载后才会完全生效' : '已卸载';
  if (!refreshSucceeded) {
    message.warning(`${resultText}；列表刷新失败，请点击刷新确认状态`);
  } else if (reloadNeeded) {
    message.warning(`${resultText}，请点击顶部“重载”按钮处理`);
  } else {
    message.success(resultText);
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

async function reloadPendingPackages() {
  const pending = pendingReloadPackages.value;
  const reloadAll = detachedReloadPending.value;
  if (!pending.length && !reloadAll) return;
  reloading.value = true;
  try {
    let failedCount = 0;
    if (reloadAll) {
      const { data } = await postSdApiV2ExtensionReloadAll({ throwOnError: true });
      failedCount = data.item.success ? 0 : 1;
    } else {
      const results = await Promise.allSettled(
        pending.map(pkg =>
          postSdApiV2ExtensionReload({
            body: { id: pkg.manifest.package.id },
            throwOnError: true,
          })
        )
      );
      failedCount = results.filter(result => result.status === 'rejected').length;
    }
    await packagesQuery.refetch();
    if (failedCount) {
      message.warning(`${failedCount} 个扩展包重载失败，请稍后重试`);
    } else {
      detachedReloadPending.value = false;
      message.success('扩展包已重载');
    }
  } catch (error) {
    handleError(error, '重载扩展包失败');
  } finally {
    reloading.value = false;
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
        const reloadNeeded = packageNeedsReload(pkg.manifest.package.id);
        await postSdApiV2ExtensionUninstall({
          body: { id: pkg.manifest.package.id, mode: 'full' },
          throwOnError: true,
        });
        const refreshSucceeded = await refreshInstallState(true);
        notifyUninstallResult(reloadNeeded, refreshSucceeded);
      } catch (error) {
        handleError(error, '卸载失败');
      }
    },
  });
}

async function handleReloadSelect(key: string) {
  if (key === 'pending') {
    await reloadPendingPackages();
    return;
  }
  try {
    reloading.value = true;
    let success = true;
    if (key === 'all') {
      const { data } = await postSdApiV2ExtensionReloadAll({ throwOnError: true });
      success = data.item.success;
    } else {
      const { data } = await postSdApiV2ExtensionReloadContent({
        body: { content: key },
        throwOnError: true,
      });
      success = data.item.success;
    }
    if (!success) {
      message.warning('重载未完成，请稍后重试');
      return;
    }
    if (key === 'all') detachedReloadPending.value = false;
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

function removeBackend(backend: StoreBackend) {
  dialog.warning({
    title: '删除仓库后端',
    content: `确认删除仓库后端「${backend.name || backend.id || backend.url}」？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
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
    },
  });
}

function handleUploadInput(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] ?? null;
  selectedUploadFile.value = file;
  uploadFileName.value = file?.name ?? '';
}

function resetInstallOptions() {
  installAutoEnable.value = true;
  installAutoReload.value = false;
}

function getInstallOptions(): InstallOptions {
  return {
    autoEnable: installAutoEnable.value,
    autoReload: installAutoReload.value && installAutoEnable.value,
  };
}

async function previewUpload() {
  if (!selectedUploadFile.value) return;
  uploadPreviewLoading.value = true;
  try {
    const { data } = await postSdApiV2ExtensionPreviewUpload({
      body: selectedUploadFile.value,
      throwOnError: true,
    });
    resetInstallOptions();
    previewSource.value = 'upload';
    previewStoreTarget.value = null;
    previewTitle.value = '上传预览';
    previewData.value = data.item;
    previewVisible.value = true;
  } catch (error) {
    handleError(error, '预览失败');
  } finally {
    uploadPreviewLoading.value = false;
  }
}

async function refreshInstallState(includeStore = false): Promise<boolean> {
  const results = await Promise.allSettled([
    packagesQuery.refetch({ throwOnError: true }),
    ...(includeStore ? [storePageQuery.refetch({ throwOnError: true })] : []),
  ]);
  const failed = results.some(result => result.status === 'rejected');
  return !failed;
}

async function runInstall(
  install: () => Promise<unknown>,
  id: string | undefined,
  options: InstallOptions,
  includeStore = false
) {
  return installPackage({
    install,
    id,
    options,
    refresh: () => refreshInstallState(includeStore),
    pendingReload: () =>
      Boolean(packages.value.find(pkg => pkg.manifest.package.id === id)?.pendingReload?.length),
  });
}
function reportInstallResult(action: string, options: InstallOptions, result: InstallResult) {
  if (result.failedStage) {
    const fallback = result.failedStage === 'enable' ? '安装后自动启用失败' : '安装后自动重载失败';
    if (result.error) {
      if (isTestModeApiError(result.error)) {
        handleError(result.error, fallback);
      } else if (!isApiErrorHandled(result.error)) {
        message.error(`${action}成功，但${getErrorMessage(result.error, fallback)}，请稍后重试`);
      }
    } else {
      message.warning(`${action}成功，但${result.message || fallback}，请稍后重试`);
    }
    return;
  }

  let successText: string;
  if (!options.autoEnable) {
    successText = `${action}成功`;
  } else if (!options.autoReload && result.reloadNeeded !== false) {
    successText = `${action}成功并已启用，等待重载生效`;
  } else if (!options.autoReload) {
    successText = `${action}成功并已启用`;
  } else {
    successText = `${action}成功、已启用并完成重载`;
  }
  if (result.refreshFailed) {
    message.warning(`${successText}，但列表刷新失败，请点击刷新确认状态`);
  } else {
    message.success(successText);
  }
}

async function installUpload(options: InstallOptions): Promise<boolean> {
  if (!selectedUploadFile.value) return false;
  const id = previewData.value?.manifest.package.id;
  uploadInstallLoading.value = true;
  try {
    const file = selectedUploadFile.value;
    const result = await runInstall(() => uploads.upload(file, 'extension-install'), id, options);
    reportInstallResult('安装', options, result);
    return true;
  } catch (error) {
    handleError(error, '上传安装失败');
    return false;
  } finally {
    uploadInstallLoading.value = false;
  }
}

async function installByUrl(url: string, options: InstallOptions): Promise<boolean> {
  if (!url) return false;
  const id = previewData.value?.manifest.package.id;
  installUrlLoading.value = true;
  try {
    const result = await runInstall(
      () =>
        postSdApiV2ExtensionInstallUrl({
          body: { url },
          throwOnError: true,
        }),
      id,
      options
    );
    installUrlInput.value = '';
    reportInstallResult('安装', options, result);
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
    resetInstallOptions();
    previewSource.value = 'url';
    previewStoreTarget.value = null;
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

function openStoreDetail(pkg: StorePackage) {
  currentStorePackage.value = pkg;
  storeDetailVisible.value = true;
}

function openStoreInstall(pkg: StorePackage) {
  resetInstallOptions();
  previewSource.value = 'store';
  previewStoreTarget.value = pkg;
  previewUrlTarget.value = '';
  previewTitle.value = `${pkg.installed ? '重装' : '安装'}扩展包`;
  previewData.value = null;
  storeDetailVisible.value = false;
  previewVisible.value = true;
}

function confirmStoreUninstall(pkg: StorePackage) {
  dialog.warning({
    title: '卸载扩展包',
    content: `确认卸载「${pkg.name}」吗？卸载后扩展包文件将被删除。`,
    positiveText: '卸载',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const reloadNeeded = packageNeedsReload(pkg.id);
        await postSdApiV2ExtensionUninstall({
          body: { id: pkg.id, mode: 'full' },
          throwOnError: true,
        });
        storeDetailVisible.value = false;
        const refreshSucceeded = await refreshInstallState(true);
        notifyUninstallResult(reloadNeeded, refreshSucceeded);
      } catch (error) {
        handleError(error, '卸载失败');
      }
    },
  });
}

async function installStorePackage(pkg: StorePackage, options: InstallOptions): Promise<boolean> {
  try {
    const result = await runInstall(
      () =>
        postSdApiV2ExtensionStoreDownload({
          body: { id: pkg.id, version: pkg.version },
          throwOnError: true,
        }),
      pkg.id,
      options,
      true
    );
    reportInstallResult(pkg.installed ? '重装' : '安装', options, result);
    return true;
  } catch (error) {
    handleError(error, '商店安装失败');
    return false;
  }
}

async function confirmPreviewInstall() {
  if (!previewData.value && !previewStoreTarget.value) return;
  installPreviewConfirmLoading.value = true;
  try {
    const options = getInstallOptions();
    let installed = false;
    if (previewSource.value === 'store' && previewStoreTarget.value) {
      installed = await installStorePackage(previewStoreTarget.value, options);
    } else if (previewSource.value === 'upload' && selectedUploadFile.value) {
      installed = await installUpload(options);
    } else if (previewSource.value === 'url' && previewUrlTarget.value) {
      installed = await installByUrl(previewUrlTarget.value, options);
    }
    if (installed) {
      previewVisible.value = false;
      previewData.value = null;
      previewStoreTarget.value = null;
      previewSource.value = 'none';
    }
  } finally {
    installPreviewConfirmLoading.value = false;
  }
}

function formatStoreFileSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '-';
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatUpdateTime(pkg: StorePackage): string {
  const timestamp = pkg.download?.updateTime ?? 0;
  return timestamp ? new Date(timestamp * 1000).toLocaleString() : '-';
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
  if (error instanceof DOMException && error.name === 'AbortError') {
    message.info(error.message);
    return;
  }
  if (isTestModeApiError(error)) {
    message.warning(getTestModeBlockMessage(error));
    return;
  }
  notifyError(error, message, fallback);
}
</script>

<style scoped>
.package-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.package-store-description {
  margin: var(--sd-space-md) 0 0;
  color: var(--sd-text-secondary);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.install-options {
  min-width: 0;
}

.install-options :deep(.n-checkbox) {
  min-height: 28px;
}

.package-pagination {
  align-self: flex-end;
}

.store-category-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sd-space-xs);
}

.store-category-label {
  flex: 0 0 auto;
}

.package-backend-add {
  display: flex;
  width: min(100%, 42rem);
  align-items: center;
  gap: var(--sd-space-xs);
}

.package-backend-url {
  min-width: 0;
  overflow-wrap: anywhere;
}

.package-setting-groups {
  display: grid;
  gap: var(--sd-space-2xs);
}

.package-install-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sd-space-md);
}

@media (max-width: 760px) {
  .package-backend-add {
    align-items: stretch;
    flex-direction: column;
  }

  .package-install-groups {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--sd-space-2xs);
  }
}
</style>
