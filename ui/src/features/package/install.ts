import { postSdApiV2ExtensionEnable, postSdApiV2ExtensionReload } from '@/api';

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
