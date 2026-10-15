import { expect, it, vi, beforeEach } from 'vitest';
import { installPackage } from './install';
import { postSdApiV2ExtensionEnable, postSdApiV2ExtensionReload } from '@/api';

vi.mock('@/api', () => ({
  postSdApiV2ExtensionEnable: vi.fn(),
  postSdApiV2ExtensionReload: vi.fn(),
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

it('honors disabled post-actions and distinguishes cache refresh failure', async () => {
  const { input } = fixture();
  input.options.autoEnable = false;
  input.refresh = async () => false;
  expect(await installPackage(input)).toEqual({
    enabled: false,
    reloaded: false,
    refreshFailed: true,
  });
  expect(postSdApiV2ExtensionEnable).not.toHaveBeenCalled();
});
