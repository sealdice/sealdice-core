import { expect, it, vi } from 'vitest';

const putSdApiV2BaseSettingValue = vi.fn(async () => ({ data: { item: { success: true } } }));

vi.mock('@/api', () => ({
  putSdApiV2BaseSettingValue,
  getSdApiV2BaseLoginSalt: vi.fn(),
  postSdApiV2BaseSettingMailTest: vi.fn(),
  postSdApiV2BaseSettingUpgrade: vi.fn(),
}));

// 后端 SetValue 的 Body 是 map[string]any，huma 会把整个请求体解成这个 map。
// 再包一层 body 的话补丁键全部落到 body 之下，applyPatch 一个都读不到 ——
// 请求返回成功，但设置没有变化。
it('保存请求把补丁作为请求体直接发出，不再多包一层 body', async () => {
  const { saveBaseSettingValue } = await import('./mutations');

  await saveBaseSettingValue({ trayTooltip: '海豹提示' });

  expect(putSdApiV2BaseSettingValue).toHaveBeenCalledWith(
    expect.objectContaining({ body: { trayTooltip: '海豹提示' } })
  );
});
