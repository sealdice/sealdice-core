// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { nextTick, type Component } from 'vue';
import { shallowMount, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import type { InternalAxiosRequestConfig } from 'axios';
import { NButton, NRadioGroup } from 'naive-ui';
import {
  client,
  getSdApiV2ImconnectionByIdConfigQueryKey,
  getSdApiV2ImconnectionSignInfoQueryKey,
  type EndPointInfo,
} from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import { useRealtimeConnectionsStore } from '@/features/connect/realtimeStore';
import ConnectPage from './connect.vue';
import ConnectCreateWizard from '@/components/connect/ConnectCreateWizard.vue';
import ConnectEditDialog from '@/components/connect/ConnectEditDialog.vue';
import ConnectAccountCard from '@/components/connect/ConnectAccountCard.vue';
import DynamicForm from '@/components/shared/DynamicForm.vue';

const feedback = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn(), warning: vi.fn() }));
const confirm = vi.hoisted(() => vi.fn());
vi.mock('naive-ui', async original => ({
  ...(await original<typeof import('naive-ui')>()),
  useMessage: () => feedback,
  useDialog: () => ({ warning: confirm }),
}));

const originalAdapter = client.instance.defaults.adapter;
let wrapper: VueWrapper | undefined;
let queryClient: QueryClient;
const endpoint: EndPointInfo = {
  id: 'a',
  nickname: 'Account A',
  adapter: {},
  cmdExecutedLastTime: 0,
  cmdExecutedNum: 0,
  enable: true,
  groupNum: 0,
  isPublic: false,
  onlineTotalTime: 0,
  platform: 'QQ',
  protocolType: 'official',
  relWorkDir: '',
  state: 1,
  userId: 'OpenQQ:1',
};
beforeEach(() => {
  vi.stubGlobal(
    'EventSource',
    class {
      addEventListener() {}
      close() {}
    }
  );
  useAuthStore(appPinia).setAccessToken('connect-test');
  useAuthStore(appPinia).finishInitialization();
});
afterEach(async () => {
  wrapper?.unmount();
  queryClient?.clear();
  useAuthStore(appPinia).clearAccessToken();
  await nextTick();
  client.instance.defaults.adapter = originalAdapter;
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});
function mountPage(component: Component, props = {}) {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(component, {
    props,
    global: {
      plugins: [appPinia, [VueQueryPlugin, { queryClient }]],
      renderStubDefaultSlot: true,
      stubs: { Modal: { template: '<div><slot /><slot name="action" /></div>' } },
    },
  });
  return wrapper;
}
function response(config: InternalAxiosRequestConfig, item: unknown) {
  return { config, data: { item }, status: 200, statusText: 'OK', headers: {} };
}
async function click(text: string) {
  const button = wrapper!.findAllComponents(NButton).find(button => button.text().trim() === text);
  if (!button) throw new Error(wrapper!.html());
  button.vm.$emit('click');
  await nextTick();
}

it.each(['manual', 'qrcode'])('创建官方 QQ 只提交一次，由创建接口验证凭据（%s）', async mode => {
  const requests: string[] = [];
  let submitted: { platform: string; config: Record<string, unknown> } | undefined;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    requests.push(`${config.method} ${path}`);
    if (path.endsWith('/protocols'))
      return response(config, {
        items: [
          {
            id: 'qq',
            name: 'QQ',
            description: '',
            methods: [
              {
                id: 'separate',
                name: '分离',
                description: '',
                protocols: [
                  {
                    key: 'officialqq',
                    schemaKey: 'officialqq',
                    name: '官方',
                    available: true,
                    deprecated: false,
                  },
                ],
              },
            ],
          },
        ],
      });
    if (path.endsWith('/schemas')) return response(config, { officialqq: [] });
    if (path.endsWith('/imconnection/') && config.method === 'post') {
      submitted = JSON.parse(config.data);
      return response(config, endpoint);
    }
    return response(config, { runtime: {} });
  };
  mountPage(ConnectCreateWizard, { show: true });
  await vi.waitFor(() => expect(wrapper!.find('button.split-item').exists()).toBe(true));
  for (let step = 0; step < 3; step++) {
    await wrapper!.get('button.split-item').trigger('click');
    await click('下一步');
  }
  wrapper!.getComponent(DynamicForm).vm.$emit('update:modelValue', {
    appID: '10001',
    appSecret: 'secret',
    useWebhook: true,
    webhookPath: '/qq',
    webhookPort: 8099,
  });
  if (mode === 'qrcode') wrapper!.getComponent(NRadioGroup).vm.$emit('update:value', mode);
  await nextTick();
  await click('添加');
  await vi.waitFor(() => expect(wrapper!.emitted('created')).toHaveLength(1));
  expect(requests.filter(item => item.startsWith('post '))).toEqual([
    'post /sd-api/v2/imconnection/',
  ]);
  expect(submitted).toEqual({
    platform: 'officialqq',
    config: {
      appID: mode === 'manual' ? '10001' : '',
      appSecret: mode === 'manual' ? 'secret' : '',
      useWebhook: true,
      webhookPath: '/qq',
      webhookPort: 8099,
    },
  });
  expect(wrapper!.emitted('created')![0]).toEqual([endpoint, mode === 'qrcode']);
});

it('编辑弹窗接受未编辑时的刷新，但保留已填写的草稿并按目标账号保存', async () => {
  let remote = 'initial';
  let saved: unknown;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/a/config'))
      return response(config, { protocolKey: 'milky', schema: [], config: { token: remote } });
    if (config.method === 'put') {
      saved = { path, body: JSON.parse(config.data) };
      return response(config, endpoint);
    }
    return response(config, { runtime: {} });
  };
  mountPage(ConnectEditDialog, { endpoint });
  const form = wrapper!.getComponent(DynamicForm);
  await vi.waitFor(() => expect(form.props('modelValue').token).toBe('initial'));
  remote = 'refreshed';
  const refresh = () =>
    queryClient.invalidateQueries({
      queryKey: getSdApiV2ImconnectionByIdConfigQueryKey({ path: { id: 'a' } }),
    });
  await refresh();
  await vi.waitFor(() => expect(form.props('modelValue').token).toBe('refreshed'));
  form.vm.$emit('update:modelValue', { token: 'draft' });
  await nextTick();
  remote = 'changed remotely';
  await refresh();
  expect(form.props('modelValue').token).toBe('draft');
  await click('保存');
  await vi.waitFor(() =>
    expect(saved).toEqual({ path: '/sd-api/v2/imconnection/a', body: { token: 'draft' } })
  );
  await vi.waitFor(() => expect(wrapper!.emitted('close')).toHaveLength(1));
});

it('签名缓存初始化新表单，刷新保留手动地址，重进表单重新补默认值', async () => {
  let version = 'v1';
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/protocols'))
      return response(config, {
        items: [
          {
            id: 'qq',
            name: 'QQ',
            methods: [
              {
                id: 'builtin',
                name: '内置',
                protocols: [
                  {
                    key: 'lagrange',
                    schemaKey: 'lagrange',
                    name: 'Lagrange',
                    available: true,
                  },
                ],
              },
            ],
          },
        ],
      });
    if (path.endsWith('/schemas')) return response(config, { lagrange: [] });
    if (path.endsWith('/sign-info'))
      return response(config, {
        items: [
          {
            version,
            selected: true,
            ignored: false,
            servers: [{ name: 'default-server', selected: true, ignored: false }],
          },
        ],
      });
    return response(config, { runtime: {} });
  };
  mountPage(ConnectCreateWizard, { show: true });
  await vi.waitFor(() => expect(wrapper!.find('button.split-item').exists()).toBe(true));
  for (let step = 0; step < 3; step++) {
    await wrapper!.get('button.split-item').trigger('click');
    await click('下一步');
  }
  const form = () => wrapper!.getComponent(DynamicForm);
  await vi.waitFor(() =>
    expect(form().props('modelValue')).toMatchObject({
      signServerVersion: 'v1',
      signServerName: 'default-server',
    })
  );
  form().vm.$emit('update:modelValue', {
    signServerVersion: '自定义',
    signServerName: 'https://custom.test',
  });
  await nextTick();
  version = 'v2';
  await queryClient.invalidateQueries({ queryKey: getSdApiV2ImconnectionSignInfoQueryKey() });
  expect(form().props('modelValue')).toMatchObject({
    signServerVersion: '自定义',
    signServerName: 'https://custom.test',
  });
  await click('上一步');
  await click('下一步');
  await vi.waitFor(() =>
    expect(form().props('modelValue')).toMatchObject({
      signServerVersion: 'v2',
      signServerName: 'default-server',
    })
  );
});

it('SSE 断开时启停和删除仍刷新列表，失败与重试只影响对应账号', async () => {
  let items = [endpoint, { ...endpoint, id: 'b', nickname: 'Account B' }];
  let fail = true;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (config.method === 'put') {
      if (fail) throw new Error('enable rejected');
      items = items.map(item => (item.id === 'a' ? { ...item, enable: false } : item));
      return response(config, items[0]);
    }
    if (config.method === 'delete') {
      items = items.filter(item => item.id !== 'a');
      return response(config, true);
    }
    return response(
      config,
      path.endsWith('/imconnection/') ? { items: structuredClone(items) } : { runtime: {} }
    );
  };
  mountPage(ConnectPage);
  await vi.waitFor(() => expect(wrapper!.findAllComponents(ConnectAccountCard)).toHaveLength(2));
  const cards = () => wrapper!.findAllComponents(ConnectAccountCard);
  cards()[0]!.vm.$emit('toggleEnable', endpoint, false);
  await confirm.mock.calls.at(-1)![0].onPositiveClick();
  await nextTick();
  expect(cards()[0]!.props('actionError')).toBe('enable rejected');
  expect(cards()[1]!.props('actionError')).toBe('');
  fail = false;
  cards()[0]!.vm.$emit('toggleEnable', endpoint, false);
  await confirm.mock.calls.at(-1)![0].onPositiveClick();
  await vi.waitFor(() => expect(cards()[0]!.props('endpoint').enable).toBe(false));
  expect(cards()[0]!.props('actionError')).toBe('');
  cards()[0]!.vm.$emit('delete', endpoint);
  await confirm.mock.calls.at(-1)![0].onPositiveClick();
  await vi.waitFor(() => expect(cards()).toHaveLength(1));
  expect(useRealtimeConnectionsStore(appPinia).connections[0]?.id).toBe('b');
});
