// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { defineComponent, ref } from 'vue';
import { createPinia, disposePinia, type Pinia } from 'pinia';
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import type { InternalAxiosRequestConfig } from 'axios';
import { client, getSdApiV2JsByNameDataInfoQueryKey } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { useJsData } from './useJsData';

const originalAdapter = client.instance.defaults.adapter;
let wrapper: VueWrapper | undefined;
let queryClient: QueryClient;
let pinia: Pinia;
afterEach(() => {
  wrapper?.unmount();
  queryClient?.clear();
  if (pinia) {
    useAuthStore(pinia).clearAccessToken();
    disposePinia(pinia);
  }
  client.instance.defaults.adapter = originalAdapter;
});

function response(config: InternalAxiosRequestConfig, item: unknown) {
  return { data: { item }, status: 200, statusText: 'OK', headers: {}, config };
}

function fixture() {
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('js-data-test');
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  const selectedPlugin = ref('a');
  let data!: ReturnType<typeof useJsData>;
  wrapper = mount(
    defineComponent({
      setup() {
        data = useJsData({
          selectedPlugin,
          dataPage: ref({ page: 1, pageSize: 20 }),
          dataKeyword: ref(''),
        });
        return () => null;
      },
    }),
    { global: { plugins: [pinia, [VueQueryPlugin, { queryClient }]] } }
  );
  return { data, selectedPlugin };
}

it('切换插件取消旧列表请求，迟到的响应不会覆盖当前插件', async () => {
  let oldRequest: InternalAxiosRequestConfig | undefined;
  let release!: () => void;
  client.instance.defaults.adapter = async config => {
    if (config.url?.includes('/a/data/list?')) {
      oldRequest = config;
      await new Promise<void>(resolve => {
        release = resolve;
      });
      return response(config, { keys: [{ key: 'a', value: 'old' }], total: 1 });
    }
    if (config.url?.includes('/b/data/list?'))
      return response(config, { keys: [{ key: 'b', value: 'new' }], total: 1 });
    return response(config, { data: [], keyCount: 0 });
  };
  const { data, selectedPlugin } = fixture();
  await vi.waitFor(() => expect(oldRequest).toBeDefined());
  selectedPlugin.value = 'b';
  await vi.waitFor(() => expect(data.dataListQuery.data.value?.keys?.[0]?.key).toBe('b'));
  expect(oldRequest!.signal?.aborted).toBe(true);
  release();
  await flushPromises();
  expect(data.dataListQuery.data.value?.keys?.[0]?.key).toBe('b');
});

it.each(['set', 'delete', 'shrink'] as const)(
  '插件切换后，%s 完成仍只失效原插件缓存',
  async operation => {
    let release!: () => void;
    const requests: InternalAxiosRequestConfig[] = [];
    client.instance.defaults.adapter = async config => {
      requests.push(config);
      if (config.method === 'post')
        await new Promise<void>(resolve => {
          release = resolve;
        });
      return response(config, { data: [], keys: [], total: 0, keyCount: 0, success: true });
    };
    const { data, selectedPlugin } = fixture();
    await vi.waitFor(() => expect(data.dataInfoQuery.isSuccess.value).toBe(true));
    const path = { name: 'a' };
    const mutation =
      operation === 'set'
        ? data.setMutation.mutateAsync({ path, body: { key: 'key', value: 'value' } })
        : operation === 'delete'
          ? data.deleteMutation.mutateAsync({ path, body: { keys: ['key'] } })
          : data.shrinkMutation.mutateAsync({ path });
    await vi.waitFor(() => expect(release).toBeDefined());
    selectedPlugin.value = 'b';
    await vi.waitFor(() =>
      expect(requests.some(request => request.url?.endsWith('/b/data/info'))).toBe(true)
    );
    await vi.waitFor(() => expect(data.dataInfoQuery.isSuccess.value).toBe(true));
    release();
    await mutation;
    const state = (name: string) =>
      queryClient.getQueryState(getSdApiV2JsByNameDataInfoQueryKey({ path: { name } }));
    expect(state('a')?.isInvalidated).toBe(true);
    expect(state('b')?.isInvalidated).toBe(false);
    expect(requests.filter(request => request.url?.endsWith('/b/data/info'))).toHaveLength(1);
    expect(requests.find(request => request.method === 'post')?.url).toContain('/a/data');
  }
);
