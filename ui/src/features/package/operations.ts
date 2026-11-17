import { computed, ref, shallowRef } from 'vue';
import {
  deleteSdApiV2ExtensionStoreBackends,
  postSdApiV2ExtensionDisable,
  postSdApiV2ExtensionEnable,
  postSdApiV2ExtensionInstallUrl,
  postSdApiV2ExtensionReload,
  postSdApiV2ExtensionReloadAll,
  postSdApiV2ExtensionReloadContent,
  postSdApiV2ExtensionStoreBackends,
  postSdApiV2ExtensionStoreBackendsDisable,
  postSdApiV2ExtensionStoreBackendsEnable,
  postSdApiV2ExtensionStoreDownload,
  postSdApiV2ExtensionUninstall,
  type PackageUploadPreview,
  type StoreBackend,
  type StorePackage,
} from '@/api';

export type InstallOptions = { autoEnable: boolean; autoReload: boolean };
export type InstallResult = {
  enabled: boolean;
  reloaded: boolean;
  reloadNeeded?: boolean;
  refreshFailed?: boolean;
  failedStage?: 'enable' | 'reload';
  error?: unknown;
  message?: string;
};

// 安装预览携带最终安装来源的快照，不再读取会变动的文件选择器。
export type InstallPreviewTarget =
  | { kind: 'upload'; file: File; data: PackageUploadPreview }
  | { kind: 'url'; url: string; data: PackageUploadPreview }
  | { kind: 'store'; pkg: StorePackage };

export function useInstallPreview() {
  const current = shallowRef<InstallPreviewTarget | null>(null);
  const open = ref(false);
  let revision = 0;

  function close(): void {
    revision += 1;
    open.value = false;
    current.value = null;
  }

  const visible = computed({
    get: () => open.value,
    set: (value: boolean) => {
      if (!value) close();
      else open.value = true;
    },
  });

  async function openAsync(load: () => Promise<InstallPreviewTarget>): Promise<boolean> {
    const requested = ++revision;
    try {
      const next = await load();
      if (requested !== revision) return false;
      current.value = next;
      open.value = true;
      return true;
    } catch (error) {
      if (requested !== revision) return false;
      throw error;
    }
  }

  function openStore(pkg: StorePackage): void {
    revision += 1;
    current.value = { kind: 'store', pkg };
    open.value = true;
  }

  return { current, visible, openAsync, openStore, close };
}

export function requirePackageActionSuccess<T extends { success: boolean; message?: string }>(
  item: T,
  fallback: string
): T {
  if (!item.success) throw new Error(item.message || fallback);
  return item;
}

// One use case for upload, URL and store installs. Only transport and the cache
// projection differ; partial post-install failures must not be called "install failed".
export async function installPackage(input: {
  install: () => Promise<unknown>;
  id?: string;
  options: InstallOptions;
  refresh: () => Promise<boolean>;
  pendingReload: () => boolean;
}): Promise<InstallResult> {
  await input.install(); // Installation/cancellation errors belong to the caller.
  const result: InstallResult = { enabled: false, reloaded: false };
  const refresh = async () => {
    try {
      if (!(await input.refresh())) result.refreshFailed = true;
    } catch {
      result.refreshFailed = true;
    }
  };
  await refresh();
  if (!input.options.autoEnable) return result;
  if (!input.id) return { ...result, failedStage: 'enable', message: '未找到包 ID' };

  let stage: 'enable' | 'reload' = 'enable';
  try {
    const enabled = await postSdApiV2ExtensionEnable({
      body: { id: input.id },
      throwOnError: true,
    });
    if (!enabled.data.item.success) {
      return { ...result, failedStage: stage, message: enabled.data.item.message };
    }
    result.enabled = true;
    await refresh();
    result.reloadNeeded = enabled.data.item.reloadNeeded || input.pendingReload();
    if (!input.options.autoReload) return result;

    stage = 'reload';
    const reloaded = await postSdApiV2ExtensionReload({
      body: { id: input.id },
      throwOnError: true,
    });
    if (!reloaded.data.item.success) {
      return { ...result, failedStage: stage, message: reloaded.data.item.message };
    }
    result.reloaded = true;
    await refresh();
    return result;
  } catch (error) {
    return { ...result, failedStage: stage, error };
  }
}
export async function installPackageFromUrl(url: string) {
  const { data } = await postSdApiV2ExtensionInstallUrl({
    body: { url },
    throwOnError: true,
  });
  return requirePackageActionSuccess(data.item, 'URL 安装失败');
}

export async function installStorePackageById(id: string, version: string) {
  const { data } = await postSdApiV2ExtensionStoreDownload({
    body: { id, version },
    throwOnError: true,
  });
  if (data.item.status !== 'installed') {
    throw new Error(data.item.message || '商店安装未完成');
  }
  return data.item;
}

// 所有扩展包写操作都在同一 API 边界检查业务 success；页面只负责确认、提示和刷新。
export async function setPackageEnabled(id: string, enabled: boolean) {
  const options = { body: { id }, throwOnError: true as const };
  const { data } = enabled
    ? await postSdApiV2ExtensionEnable(options)
    : await postSdApiV2ExtensionDisable(options);
  return requirePackageActionSuccess(data.item, enabled ? '启用失败' : '禁用失败');
}

export async function reloadPackageById(id: string) {
  const { data } = await postSdApiV2ExtensionReload({ body: { id }, throwOnError: true });
  return requirePackageActionSuccess(data.item, '重载失败');
}

export async function reloadAllPackages() {
  const { data } = await postSdApiV2ExtensionReloadAll({ throwOnError: true });
  return requirePackageActionSuccess(data.item, '重载全部扩展包失败');
}

export async function reloadPackageContent(content: string) {
  const { data } = await postSdApiV2ExtensionReloadContent({
    body: { content },
    throwOnError: true,
  });
  return requirePackageActionSuccess(data.item, '重载扩展内容失败');
}

export async function uninstallPackageById(id: string) {
  const { data } = await postSdApiV2ExtensionUninstall({
    body: { id, mode: 'full' },
    throwOnError: true,
  });
  return requirePackageActionSuccess(data.item, '卸载失败');
}

export async function addPackageBackend(url: string) {
  const { data } = await postSdApiV2ExtensionStoreBackends({
    body: { url },
    throwOnError: true,
  });
  return requirePackageActionSuccess(data.item, '添加后端失败');
}

function backendBody(backend: StoreBackend) {
  return { id: backend.id || '', backendID: backend.id || '', url: backend.url || '' };
}

export async function setPackageBackendEnabled(backend: StoreBackend, enabled: boolean) {
  const options = { body: backendBody(backend), throwOnError: true as const };
  const { data } = enabled
    ? await postSdApiV2ExtensionStoreBackendsEnable(options)
    : await postSdApiV2ExtensionStoreBackendsDisable(options);
  return requirePackageActionSuccess(data.item, enabled ? '启用仓库后端失败' : '禁用仓库后端失败');
}

export async function removePackageBackend(backend: StoreBackend) {
  const { data } = await deleteSdApiV2ExtensionStoreBackends({
    body: backendBody(backend),
    throwOnError: true,
  });
  return requirePackageActionSuccess(data.item, '删除仓库后端失败');
}
