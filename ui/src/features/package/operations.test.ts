import { expect, it, vi, beforeEach } from 'vitest';
import {
  addPackageBackend,
  installPackage,
  installPackageFromUrl,
  installStorePackageById,
  reloadPackageById,
  requirePackageActionSuccess,
  setPackageEnabled,
  useInstallPreview,
} from './operations';
import {
  postSdApiV2ExtensionEnable,
  postSdApiV2ExtensionInstallUrl,
  postSdApiV2ExtensionStoreDownload,
  postSdApiV2ExtensionDisable,
  postSdApiV2ExtensionReload,
  postSdApiV2ExtensionStoreBackends,
} from '@/api';

vi.mock('@/api', () => ({
  postSdApiV2ExtensionEnable: vi.fn(),
  postSdApiV2ExtensionReload: vi.fn(),
  postSdApiV2ExtensionDisable: vi.fn(),
  postSdApiV2ExtensionStoreBackends: vi.fn(),
  postSdApiV2ExtensionInstallUrl: vi.fn(),
  postSdApiV2ExtensionStoreDownload: vi.fn(),
}));
beforeEach(() => vi.resetAllMocks());

function fixture() {
  const calls: string[] = [];
  vi.mocked(postSdApiV2ExtensionEnable).mockImplementation(async () => {
    calls.push('enable');
    return { data: { item: { success: true, reloadNeeded: true } } } as never;
  });
  vi.mocked(postSdApiV2ExtensionReload).mockImplementation(async () => {
    calls.push('reload');
    return { data: { item: { success: true } } } as never;
  });
  const input = {
    id: 'test/pkg',
    options: { autoEnable: true, autoReload: true },
    install: async () => {
      calls.push('install');
    },
    refresh: async () => {
      calls.push('refresh');
      return true;
    },
    pendingReload: () => false,
  };
  return { input, calls };
}

it('installs, enables and reloads in order, refreshing after each completed change', async () => {
  const { input, calls } = fixture();
  expect(await installPackage(input)).toMatchObject({ enabled: true, reloaded: true });
  expect(calls).toEqual(['install', 'refresh', 'enable', 'refresh', 'reload', 'refresh']);
});

it('reports an enable failure as partial success and never reloads', async () => {
  const { input } = fixture();
  vi.mocked(postSdApiV2ExtensionEnable).mockRejectedValue(new Error('enable failed'));
  expect(await installPackage(input)).toMatchObject({ enabled: false, failedStage: 'enable' });
  expect(postSdApiV2ExtensionReload).not.toHaveBeenCalled();
});

it('preserves successful installation and enablement when reload fails', async () => {
  const { input } = fixture();
  vi.mocked(postSdApiV2ExtensionReload).mockResolvedValue({
    data: { item: { success: false, message: 'busy' } },
  } as never);
  expect(await installPackage(input)).toMatchObject({
    enabled: true,
    reloaded: false,
    failedStage: 'reload',
    message: 'busy',
  });
});

it('does not run post-install actions after cancellation', async () => {
  const { input } = fixture();
  input.install = async () => {
    throw new DOMException('Cancelled', 'AbortError');
  };
  await expect(installPackage(input)).rejects.toMatchObject({ name: 'AbortError' });
  expect(postSdApiV2ExtensionEnable).not.toHaveBeenCalled();
  expect(postSdApiV2ExtensionReload).not.toHaveBeenCalled();
});

it.each([false, true])('保留安装成功状态并标记刷新失败（抛异常=%s）', async throws => {
  const { input } = fixture();
  input.options.autoEnable = false;
  input.refresh = async () => {
    if (throws) throw new Error('刷新请求失败');
    return false;
  };
  expect(await installPackage(input)).toEqual({
    enabled: false,
    reloaded: false,
    refreshFailed: true,
  });
  expect(postSdApiV2ExtensionEnable).not.toHaveBeenCalled();
});
it('拒绝被 HTTP 200 包装的业务失败', () => {
  expect(() =>
    requirePackageActionSuccess({ success: false, message: '启用失败' }, '操作失败')
  ).toThrow('启用失败');
  expect(requirePackageActionSuccess({ success: true }, '操作失败')).toEqual({ success: true });
});

it('旧预览完成或关闭后不能覆盖最新安装来源', async () => {
  const preview = useInstallPreview();
  const upload = { kind: 'upload', file: {} as File, data: {} as never } as const;
  const store = { kind: 'store', pkg: { id: 'latest' } as never } as const;
  let completeOld!: (target: typeof upload) => void;
  const pending = preview.openAsync(
    () =>
      new Promise(resolve => {
        completeOld = resolve;
      })
  );
  preview.openStore(store.pkg);
  completeOld(upload);
  expect(await pending).toBe(false);
  expect(preview.current.value).toEqual(store);
  preview.close();
  expect(preview.current.value).toBeNull();
  expect(preview.visible.value).toBe(false);
});

it('关闭弹窗会作废未完成的异步预览与旧请求错误', async () => {
  const preview = useInstallPreview();
  let rejectOld!: (reason: Error) => void;
  const pending = preview.openAsync(
    () =>
      new Promise((_resolve, reject) => {
        rejectOld = reject;
      })
  );
  preview.close();
  rejectOld(new Error('stale preview failed'));
  expect(await pending).toBe(false);
  expect(preview.visible.value).toBe(false);
});
it('HTTP 200 内部的启用失败仍按失败处理，避免误报成功', async () => {
  vi.mocked(postSdApiV2ExtensionEnable).mockResolvedValue({
    data: { item: { success: false, message: '扩展包未满足条件' } },
  } as never);
  await expect(setPackageEnabled('test/pkg', true)).rejects.toThrow('扩展包未满足条件');
});

it('禁用和重载均校验业务结果，而不只看 HTTP 状态', async () => {
  vi.mocked(postSdApiV2ExtensionDisable).mockResolvedValue({
    data: { item: { success: true } },
  } as never);
  await expect(setPackageEnabled('test/pkg', false)).resolves.toMatchObject({ success: true });
  expect(postSdApiV2ExtensionDisable).toHaveBeenCalledWith(
    expect.objectContaining({ body: { id: 'test/pkg' } })
  );
  vi.mocked(postSdApiV2ExtensionReload).mockResolvedValue({
    data: { item: { success: false, message: '重载失败' } },
  } as never);
  await expect(reloadPackageById('test/pkg')).rejects.toThrow('重载失败');
});

it('仓库操作拒绝业务失败状态', async () => {
  vi.mocked(postSdApiV2ExtensionStoreBackends).mockResolvedValue({
    data: { item: { success: false, message: '后端已存在' } },
  } as never);
  await expect(addPackageBackend('https://example.org')).rejects.toThrow('后端已存在');
});

it('URL 安装返回业务失败时不执行启用或重载', async () => {
  const { input } = fixture();
  vi.mocked(postSdApiV2ExtensionInstallUrl).mockResolvedValue({
    data: { item: { success: false, message: '包文件校验失败' } },
  } as never);
  input.install = async () => {
    await installPackageFromUrl('https://example.org/sample.sealpack');
  };
  await expect(installPackage(input)).rejects.toThrow('包文件校验失败');
  expect(postSdApiV2ExtensionEnable).not.toHaveBeenCalled();
  expect(postSdApiV2ExtensionReload).not.toHaveBeenCalled();
});

it('商店安装只把真正已安装的状态视为成功', async () => {
  vi.mocked(postSdApiV2ExtensionStoreDownload).mockResolvedValue({
    data: { item: { id: 'test/pkg', version: '1.0.0', status: 'skipped', message: '已跳过' } },
  } as never);
  await expect(installStorePackageById('test/pkg', '1.0.0')).rejects.toThrow('已跳过');
  vi.mocked(postSdApiV2ExtensionStoreDownload).mockResolvedValue({
    data: { item: { id: 'test/pkg', version: '1.0.0', status: 'installed' } },
  } as never);
  await expect(installStorePackageById('test/pkg', '1.0.0')).resolves.toMatchObject({
    status: 'installed',
  });
});
