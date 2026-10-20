// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { nextTick, type Component } from 'vue';
import { createPinia, disposePinia, type Pinia } from 'pinia';
import { mount, shallowMount, type VueWrapper } from '@vue/test-utils';
import { NInputNumber } from 'naive-ui';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import type { InternalAxiosRequestConfig } from 'axios';
import { client, type CensorConfigBody } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { useUnsavedChangesStore } from '@/features/unsavedChanges/store';
import { createDefaultCensorConfig } from '@/features/censor/viewModel';
import CensorConfigView from '@/components/censor/CensorConfigView.vue';
import HelpdocConfigDialog from '@/components/helpdoc/HelpdocConfigDialog.vue';
import CensorPage from './censor.vue';
import HelpdocPage from './helpdoc.vue';

// 搜索表单位于 shallow stub 中；不加载其独立打包的 UI 运行时。
vi.mock('pro-naive-ui', () => ({ createProSearchForm: vi.fn() }));

vi.mock('naive-ui', async original => ({
  ...(await original<typeof import('naive-ui')>()),
  useMessage: () => ({ success: vi.fn(), warning: vi.fn(), error: vi.fn() }),
  useDialog: () => ({ warning: vi.fn() }),
}));

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

function fixture(page: Component) {
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('config-flow-test');
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(page, {
    global: { plugins: [pinia, [VueQueryPlugin, { queryClient }]], renderStubDefaultSlot: true },
  });
  return useUnsavedChangesStore(pinia);
}

function response(config: InternalAxiosRequestConfig, item: unknown) {
  return { data: { item }, config, status: 200, statusText: 'OK', headers: {} };
}

it('拦截表单及嵌套等级只提交新配置，不修改父级配置', async () => {
  const config = createDefaultCensorConfig();
  const before = structuredClone(config);
  wrapper = mount(CensorConfigView, { props: { config, saving: false, modified: false } });
  await wrapper.get('.censor-regex-input input').setValue('new regex');
  expect(wrapper.emitted('update:config')?.at(-1)).toEqual([
    { ...config, filterRegex: 'new regex' },
  ]);
  wrapper.findAllComponents(NInputNumber)[0]!.vm.$emit('update:value', 3);
  expect(wrapper.emitted('update:config')?.at(-1)).toEqual([
    {
      ...config,
      filterRegex: 'new regex',
      levelConfig: {
        ...config.levelConfig,
        notice: { ...config.levelConfig.notice, threshold: 3 },
      },
    },
  ]);
  expect(config).toEqual(before);
});

it.each([false, true])(
  '拦截配置保存保留提交快照，放弃只还原未提交编辑（继续编辑=%s）',
  async edits => {
    let remote = { ...createDefaultCensorConfig(), filterRegex: 'remote' };
    let submitted: CensorConfigBody | undefined;
    let finish!: () => void;
    client.instance.defaults.adapter = async config => {
      const path = new URL(config.url!, 'http://local').pathname;
      if (path.endsWith('/censor/status')) return response(config, { enable: true });
      if (path.endsWith('/censor/config') && config.method === 'post') {
        submitted = JSON.parse(config.data);
        await new Promise<void>(resolve => {
          finish = resolve;
        });
        remote = submitted!;
        return response(config, { success: true });
      }
      if (path.endsWith('/censor/config')) return response(config, structuredClone(remote));
      return response(config, { data: [], total: 0 });
    };
    const store = fixture(CensorPage);
    await vi.waitFor(() => expect(wrapper!.findComponent(CensorConfigView).exists()).toBe(true));
    const form = wrapper!.getComponent(CensorConfigView);
    await vi.waitFor(() => expect(form.props('config').filterRegex).toBe('remote'));
    form.vm.$emit('update:config', { ...form.props('config'), filterRegex: 'submitted' });
    await nextTick();
    const saved = store.runPendingAction('censor-config');
    await vi.waitFor(() => expect(finish).toBeDefined());
    if (edits) {
      form.vm.$emit('update:config', { ...form.props('config'), filterRegex: 'later edit' });
      await nextTick();
    }
    finish();
    await saved;
    expect(submitted?.filterRegex).toBe('submitted');
    expect(form.props('config').filterRegex).toBe(edits ? 'later edit' : 'submitted');
    expect(store.hasUnsavedChanges).toBe(edits);
    expect(store.activePendingActions.some(action => action.kind === 'reload')).toBe(true);
    if (edits) {
      await store.discardPendingAction('censor-config');
      expect(form.props('config').filterRegex).toBe('submitted');
      expect(store.hasUnsavedChanges).toBe(false);
    }
  }
);

it.each([false, true])('帮助文档保存后刷新不会覆盖新添加的别名（继续编辑=%s）', async edits => {
  let remote = { default: ['old'] };
  let submitted: { aliases: typeof remote } | undefined;
  let finish!: () => void;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/helpdoc/config') && config.method === 'post') {
      submitted = JSON.parse(config.data);
      await new Promise<void>(resolve => {
        finish = resolve;
      });
      remote = submitted!.aliases;
      return response(config, { success: true });
    }
    if (path.endsWith('/helpdoc/config'))
      return response(config, { aliases: structuredClone(remote) });
    return response(config, { data: [], total: 0 });
  };
  const store = fixture(HelpdocPage);
  const form = wrapper!.getComponent(HelpdocConfigDialog);
  await vi.waitFor(() => expect(form.props('aliases').get('default')).toEqual(['old']));
  form.vm.$emit('addAlias', 'default', 'submitted');
  await nextTick();
  const saved = store.runPendingAction('helpdoc-config');
  await vi.waitFor(() => expect(finish).toBeDefined());
  if (edits) {
    form.vm.$emit('addAlias', 'default', 'later edit');
    await nextTick();
  }
  finish();
  await saved;
  expect(submitted?.aliases.default).toEqual(['old', 'submitted']);
  expect(form.props('aliases').get('default')).toEqual(
    edits ? ['old', 'submitted', 'later edit'] : ['old', 'submitted']
  );
  expect(store.hasUnsavedChanges).toBe(edits);
  if (edits) {
    await store.discardPendingAction('helpdoc-config');
    expect(form.props('aliases').get('default')).toEqual(['old', 'submitted']);
    expect(store.hasUnsavedChanges).toBe(false);
  }
});
