import { expect, it, vi } from 'vitest';

const putSdApiV2BaseSettingValue = vi.fn(async () => ({ data: { item: { success: true } } }));

vi.mock('@/api', () => ({
  putSdApiV2BaseSettingValue,
  getSdApiV2BaseLoginSalt: vi.fn(),
  postSdApiV2BaseSettingMailTest: vi.fn(),
  postSdApiV2BaseSettingUpgrade: vi.fn(),
}));

// Huma 将请求体直接绑定到补丁 DTO；SDK 的 body 参数只包装一次。
it('保存请求把补丁作为请求体直接发出，不再多包一层 body', async () => {
  const { saveBaseSettingValue } = await import('./mutations');

  await saveBaseSettingValue({ trayTooltip: '海豹提示' });

  expect(putSdApiV2BaseSettingValue).toHaveBeenCalledWith(
    expect.objectContaining({ body: { trayTooltip: '海豹提示' } })
  );
});
