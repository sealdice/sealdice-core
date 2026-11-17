// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';
import { shallowMount, type VueWrapper } from '@vue/test-utils';
import { createPinia, disposePinia, type Pinia } from 'pinia';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import type { InternalAxiosRequestConfig } from 'axios';
import { client, getSdApiV2JsConfigsQueryKey, type ConfigItem } from '@/api';
import { useAuthStore } from '@/features/auth/store';
import { useUnsavedChangesStore } from '@/features/unsavedChanges/store';
import JsConfigView from './JsConfigView.vue';
import JsConfigItemEditor from './JsConfigItemEditor.vue';

vi.mock('naive-ui', async original => ({
  ...(await original<typeof import('naive-ui')>()),
  useMessage: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
  useDialog: () => ({ warning: vi.fn() }),
}));
const originalAdapter = client.instance.defaults.adapter;
let wrapper: VueWrapper | undefined;
let queryClient: QueryClient;
let pinia: Pinia;
afterEach(() => {
  wrapper?.unmount();
  queryClient?.clear();
  if (pinia) disposePinia(pinia);
  client.instance.defaults.adapter = originalAdapter;
});
function item(key: string, value: unknown, type = 'string'): ConfigItem {
  return { key, value, type, defaultValue: 'default', description: '' };
}
function response(config: InternalAxiosRequestConfig, value: unknown) {
  return { config, data: { item: value }, status: 200, statusText: 'OK', headers: {} };
}
function mountConfig() {
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('config-test');
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(JsConfigView, {
    global: {
      plugins: [pinia, [VueQueryPlugin, { queryClient }]],
      renderStubDefaultSlot: true,
      stubs: {
        ResponsiveTabs: {
          inheritAttrs: false,
          props: ['options'],
          template: '<slot name="panel" :option="options[0]" />',
        },
      },
    },
  });
  return useUnsavedChangesStore(pinia);
}
function field(plugin: string, key = 'text') {
  return wrapper!
    .findAllComponents(JsConfigItemEditor)
    .find(
      component => component.props('pluginName') === plugin && component.props('item').key === key
    )!;
}
async function edit(plugin: string, value: unknown, key = 'text') {
  field(plugin, key).vm.$emit('change', plugin, key, value);
  await nextTick();
}

it.each([false, true])(
  '保存与刷新只确认提交快照，继续编辑后仍可放弃（刷新失败=%s）',
  async refreshFails => {
    const remote = {
      alpha: { pluginName: 'alpha', configs: [item('text', 'old')], orderedConfigKeys: ['text'] },
    };
    let finish!: () => void;
    let submitted: { name: string; config: Record<string, unknown> } | undefined;
    let saved = false;
    client.instance.defaults.adapter = async config => {
      const path = new URL(config.url!, 'http://local').pathname;
      if (path.endsWith('/js/configs') && config.method === 'post') {
        submitted = JSON.parse(config.data);
        await new Promise<void>(resolve => {
          finish = resolve;
        });
        remote.alpha.configs[0]!.value = submitted!.config.text;
        saved = true;
        return response(config, { success: true });
      }
      if (path.endsWith('/js/configs')) {
        if (saved && refreshFails) throw new Error('refresh failed');
        return response(config, structuredClone(remote));
      }
      return response(config, { runtime: {}, configs: [] });
    };
    const store = mountConfig();
    await vi.waitFor(() => expect(field('alpha')).toBeDefined());
    await edit('alpha', 'submitted');
    const saving = store.runPendingAction('js-config');
    await vi.waitFor(() => expect(finish).toBeDefined());
    await edit('alpha', 'later edit');
    await queryClient.invalidateQueries({ queryKey: getSdApiV2JsConfigsQueryKey() });
    expect(field('alpha').props('item').value).toBe('later edit');
    finish();
    expect(await saving).toBe(true);
    expect(submitted?.config.text).toBe('submitted');
    expect(field('alpha').props('item').value).toBe('later edit');
    expect(store.hasUnsavedChanges).toBe(true);
    await store.discardPendingAction('js-config');
    expect(field('alpha').props('item').value).toBe('submitted');
    expect(store.hasUnsavedChanges).toBe(false);
  }
);

it('多个插件部分保存失败时，只重试未提交的插件', async () => {
  const remote = {
    alpha: { pluginName: 'alpha', configs: [item('text', 'old A')], orderedConfigKeys: ['text'] },
    beta: { pluginName: 'beta', configs: [item('text', 'old B')], orderedConfigKeys: ['text'] },
  };
  const requests: string[] = [];
  let fail = true;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/js/configs') && config.method === 'post') {
      const body = JSON.parse(config.data) as { name: 'alpha' | 'beta'; config: { text: string } };
      requests.push(body.name);
      if (body.name === 'beta' && fail) throw new Error('save B failed');
      remote[body.name].configs[0]!.value = body.config.text;
      return response(config, { success: true });
    }
    return response(
      config,
      path.endsWith('/js/configs') ? structuredClone(remote) : { runtime: {}, configs: [] }
    );
  };
  const store = mountConfig();
  await vi.waitFor(() => expect(field('beta')).toBeDefined());
  await edit('alpha', 'new A');
  await edit('beta', 'new B');
  expect(await store.runPendingAction('js-config')).toBe(false);
  expect(store.hasUnsavedChanges).toBe(true);
  expect(field('alpha').props('item').value).toBe('new A');
  expect(field('beta').props('item').value).toBe('new B');
  fail = false;
  expect(await store.runPendingAction('js-config')).toBe(true);
  expect(requests).toEqual(['alpha', 'beta', 'beta']);
  expect(store.hasUnsavedChanges).toBe(false);
});

it('Cron 校验等待期间不能保存，迟到的旧校验不会覆盖新结果', async () => {
  const pending: Array<{ resolve: () => void; reject: () => void }> = [];
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/cron/check')) {
      await new Promise<void>((resolve, reject) =>
        pending.push({ resolve, reject: () => reject(new Error('old invalid cron')) })
      );
      return response(config, { valid: true });
    }
    return response(
      config,
      path.endsWith('/js/configs')
        ? {
            alpha: {
              pluginName: 'alpha',
              configs: [item('cron', '0 8 * * *', 'task:cron')],
              orderedConfigKeys: ['cron'],
            },
          }
        : { runtime: {}, configs: [] }
    );
  };
  const store = mountConfig();
  await vi.waitFor(() => expect(field('alpha', 'cron')).toBeDefined());
  await edit('alpha', 'bad cron', 'cron');
  await vi.waitFor(() => expect(pending).toHaveLength(1));
  await edit('alpha', '0 9 * * *', 'cron');
  await vi.waitFor(() => expect(pending).toHaveLength(2));
  expect(store.activePendingActions[0]?.canSave).toBe(false);
  pending[1]!.resolve();
  await vi.waitFor(() => expect(store.activePendingActions[0]?.canSave).toBe(true));
  pending[0]!.reject();
  await nextTick();
  expect(field('alpha', 'cron').props('errorText')).toBeUndefined();
  expect(field('alpha', 'cron').props('item').value).toBe('0 9 * * *');
});

it.each([false, true])('重置只清除成功项的草稿，保留其他编辑（重置失败=%s）', async fails => {
  const remote = {
    alpha: {
      pluginName: 'alpha',
      configs: [item('text', 'old'), item('other', 'old')],
      orderedConfigKeys: ['text', 'other'],
    },
  };
  const resets: string[] = [];
  let submitted: Record<string, unknown> | undefined;
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    if (path.endsWith('/configs/reset')) {
      resets.push('reset');
      if (fails) throw new Error('reset failed');
      remote.alpha.configs[0]!.value = 'default';
      return response(config, { success: true });
    }
    if (path.endsWith('/js/configs') && config.method === 'post') {
      submitted = JSON.parse(config.data).config;
      remote.alpha.configs.forEach(item => {
        if (Object.prototype.hasOwnProperty.call(submitted, item.key))
          item.value = submitted![item.key];
      });
      return response(config, { success: true });
    }
    return response(
      config,
      path.endsWith('/js/configs') ? structuredClone(remote) : { runtime: {}, configs: [] }
    );
  };
  const store = mountConfig();
  await vi.waitFor(() => expect(field('alpha')).toBeDefined());
  await edit('alpha', 'draft');
  await edit('alpha', 'other draft', 'other');
  expect(field('alpha', 'other').props('item').value).toBe('other draft');
  field('alpha').vm.$emit('reset', 'alpha', 'text');
  await vi.waitFor(() => expect(resets).toHaveLength(1));
  await vi.waitFor(() => expect(field('alpha').props('resetDisabled')).toBe(false));
  expect(field('alpha').props('item').value).toBe(fails ? 'draft' : 'default');
  expect(field('alpha', 'other').props('item').value).toBe('other draft');
  expect(await store.runPendingAction('js-config')).toBe(true);
  expect(submitted).toEqual(
    fails ? { text: 'draft', other: 'other draft' } : { other: 'other draft' }
  );
});
