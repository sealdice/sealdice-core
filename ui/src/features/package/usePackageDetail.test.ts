// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { defineComponent, nextTick } from 'vue';
import { createPinia } from 'pinia';
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import {
  getSdApiV2ExtensionConfig,
  getSdApiV2ExtensionConfigSchema,
  putSdApiV2ExtensionConfig,
  type Instance,
} from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { usePackageDetail } from './usePackageDetail';

vi.mock('@/api', () => ({
  getSdApiV2ExtensionConfig: vi.fn(),
  getSdApiV2ExtensionConfigSchema: vi.fn(),
  putSdApiV2ExtensionConfig: vi.fn(),
}));
function deferred() {
  let resolve!: (value: unknown) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
const reads = new Map<string, ReturnType<typeof deferred>>();
let wrapper: VueWrapper | undefined;
let client: QueryClient;
let auth: ReturnType<typeof useAuthStore>;
beforeEach(() => {
  vi.resetAllMocks();
  reads.clear();
  vi.mocked(getSdApiV2ExtensionConfig).mockImplementation(options => {
    const id = String(options?.query?.id);
    if (!reads.has(id)) reads.set(id, deferred());
    return reads.get(id)!.promise as never;
  });
  vi.mocked(getSdApiV2ExtensionConfigSchema).mockResolvedValue({
    data: { item: { value: { type: 'string' } } },
  } as never);
});
afterEach(() => {
  wrapper?.unmount();
  client?.clear();
  auth?.clearAccessToken();
  auth?.$dispose();
});
function fixture(token = 'detail-test') {
  const pinia = createPinia();
  auth = useAuthStore(pinia);
  auth.setAccessToken(token);
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onSaved = vi.fn();
  const onError = vi.fn();
  let detail!: ReturnType<typeof usePackageDetail>;
  wrapper = mount(
    defineComponent({
      setup() {
        detail = usePackageDetail({ onSaved, onError });
        return () => null;
      },
    }),
    { global: { plugins: [pinia, [VueQueryPlugin, { queryClient: client }]] } }
  );
  const open = async (id: string) => {
    detail.open({ manifest: { package: { id } } } as Instance);
    await vi.waitFor(() => expect(reads.has(id)).toBe(true));
  };
  const resolve = async (id: string) => {
    reads.get(id)!.resolve({ data: { item: { value: id } } });
    await flushPromises();
  };
  return { detail, open, resolve, onSaved, onError };
}

it.each([false, true])('切换包后忽略旧请求的结果（旧请求失败=%s）', async fails => {
  const { detail, open, resolve, onError } = fixture();
  await open('a');
  await open('b');
  await resolve('b');
  await vi.waitFor(() => expect(detail.config.value).toEqual({ value: 'b' }));
  if (fails) reads.get('a')!.reject(new Error('旧请求失败'));
  else await resolve('a');
  await flushPromises();
  expect(detail.pkg.value?.manifest.package.id).toBe('b');
  expect(detail.config.value).toEqual({ value: 'b' });
  expect(detail.visible.value).toBe(true);
  expect(onError).not.toHaveBeenCalled();
});

it('保存旧包期间切换到新包，不覆盖新包配置', async () => {
  const saved = deferred();
  vi.mocked(putSdApiV2ExtensionConfig).mockReturnValue(saved.promise as never);
  const { detail, open, resolve, onSaved } = fixture();
  await open('a');
  await resolve('a');
  detail.saveConfig({ value: 'saved-a' });
  await vi.waitFor(() => expect(putSdApiV2ExtensionConfig).toHaveBeenCalled());
  await open('b');
  await resolve('b');
  saved.resolve({ data: { item: { success: true } } });
  await vi.waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
  expect(putSdApiV2ExtensionConfig).toHaveBeenCalledWith(
    expect.objectContaining({ query: { id: 'a' }, body: { value: 'saved-a' } })
  );
  expect(detail.config.value).toEqual({ value: 'b' });
});

it('HTTP 200 内业务保存失败时不显示保存成功', async () => {
  vi.mocked(putSdApiV2ExtensionConfig).mockResolvedValue({
    data: { item: { success: false, message: '写入失败' } },
  } as never);
  const { detail, open, resolve, onSaved, onError } = fixture();
  await open('a');
  await resolve('a');
  detail.saveConfig({ value: 'bad' });
  await vi.waitFor(() =>
    expect(onError).toHaveBeenCalledWith(
      expect.objectContaining({ message: '写入失败' }),
      '保存配置失败'
    )
  );
  expect(onSaved).not.toHaveBeenCalled();
});

it('登录后才读取配置，退出登录关闭并清空当前详情', async () => {
  const { detail, resolve } = fixture('');
  detail.open({ manifest: { package: { id: 'a' } } } as Instance);
  await nextTick();
  expect(getSdApiV2ExtensionConfig).not.toHaveBeenCalled();
  auth.setAccessToken('detail-test');
  await vi.waitFor(() => expect(reads.has('a')).toBe(true));
  await resolve('a');
  expect(detail.config.value).toEqual({ value: 'a' });
  auth.clearAccessToken();
  await nextTick();
  expect(detail.visible.value).toBe(false);
  expect(detail.pkg.value).toBeNull();
  expect(detail.config.value).toBeNull();
});
