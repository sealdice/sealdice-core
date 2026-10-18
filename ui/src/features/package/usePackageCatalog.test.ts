// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { defineComponent, ref } from 'vue';
import { createPinia, disposePinia } from 'pinia';
import { mount, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { useAuthStore } from '@/features/auth/store';
import { getSdApiV2ExtensionStorePage } from '@/api';
import { usePackageCatalog } from './usePackageCatalog';

vi.mock('@/api', () => ({
  getSdApiV2ExtensionPackages: vi.fn(async () => ({ data: { item: { items: [] } } })),
  getSdApiV2ExtensionStoreBackends: vi.fn(async () => ({ data: { item: { items: [] } } })),
  getSdApiV2ExtensionStorePage: vi.fn(async () => ({ data: { item: { items: [], next: null } } })),
  postSdApiV2ExtensionPackagesRefresh: vi.fn(),
}));
let wrapper: VueWrapper;
const pinia = createPinia();
const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
afterEach(() => {
  wrapper?.unmount();
  client.clear();
  disposePinia(pinia);
});

it('未登录时不加载目录，登录和筛选变化按当前参数请求', async () => {
  const auth = useAuthStore(pinia);
  auth.clearAccessToken();
  const name = ref('first');
  wrapper = mount(
    defineComponent({
      setup() {
        usePackageCatalog(() => ({ name: name.value, pageNum: 1, pageSize: 20 }));
        return () => null;
      },
    }),
    { global: { plugins: [pinia, [VueQueryPlugin, { queryClient: client }]] } }
  );
  expect(getSdApiV2ExtensionStorePage).not.toHaveBeenCalled();
  auth.setAccessToken('test');
  await vi.waitFor(() =>
    expect(getSdApiV2ExtensionStorePage).toHaveBeenCalledWith(
      expect.objectContaining({
        query: { name: 'first', pageNum: 1, pageSize: 20 },
        signal: expect.any(AbortSignal),
      })
    )
  );
  name.value = 'second';
  await vi.waitFor(() =>
    expect(getSdApiV2ExtensionStorePage).toHaveBeenLastCalledWith(
      expect.objectContaining({ query: { name: 'second', pageNum: 1, pageSize: 20 } })
    )
  );
});
