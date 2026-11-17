// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';
import { shallowMount, type VueWrapper } from '@vue/test-utils';
import { createPinia, disposePinia, type Pinia } from 'pinia';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { useAuthStore } from '@/features/auth/store';
import { useUnsavedChangesStore } from '@/features/unsavedChanges/store';
import BaseSettingPage from '@/pages/misc/base-setting.vue';
import BaseSettingFieldRenderer from '@/components/base-setting/BaseSettingFieldRenderer.vue';

const api = vi.hoisted(() => ({ read: vi.fn(), salt: vi.fn(), save: vi.fn() }));
vi.mock('@/api', () => ({
  getSdApiV2BaseSettingValueOptions: () => ({ queryKey: ['settings'], queryFn: api.read }),
  getSdApiV2BaseSettingValueQueryKey: () => ['settings'],
  getSdApiV2BaseOverviewOptions: () => ({
    queryKey: ['overview'],
    queryFn: async () => ({ item: { runtime: {} } }),
  }),
  getSdApiV2BaseLoginSalt: api.salt,
  putSdApiV2BaseSettingValue: api.save,
  postSdApiV2BaseSettingMailTest: vi.fn(),
  postSdApiV2BaseSettingUpgrade: vi.fn(),
}));
vi.mock('naive-ui', async importOriginal => ({
  ...(await importOriginal<typeof import('naive-ui')>()),
  useMessage: () => ({ success: vi.fn(), info: vi.fn(), error: vi.fn() }),
}));

let wrapper: VueWrapper | undefined;
let client: QueryClient | undefined;
let pinia: Pinia | undefined;
afterEach(() => {
  wrapper?.unmount();
  client?.clear();
  if (pinia) disposePinia(pinia);
  vi.clearAllMocks();
});

it.each([
  [false, false],
  [true, false],
  [false, true],
  [true, true],
])('保存保留对应快照（继续编辑=%s，刷新失败=%s）', async (edits, refreshFails) => {
  const remote = {
    uiPassword: '------',
    trayTooltip: '旧提示',
    commandPrefix: ['.'],
    diceMasters: [],
    noticeIds: [],
    extDefaultSettings: [],
  };
  api.read.mockImplementation(async () => {
    if (refreshFails && api.read.mock.calls.length > 1) throw new Error('刷新失败');
    return { item: structuredClone(remote) };
  });
  let releaseSalt!: (value: unknown) => void;
  api.salt.mockImplementation(
    () =>
      new Promise(resolve => {
        releaseSalt = resolve;
      })
  );
  api.save.mockImplementation(async ({ body }) => {
    Object.assign(remote, body, { uiPassword: '------' });
    return { data: { item: { success: true } } };
  });
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('settings-test');
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(BaseSettingPage, {
    global: {
      plugins: [pinia, [VueQueryPlugin, { queryClient: client }]],
      renderStubDefaultSlot: true,
      stubs: {
        ResponsiveTabs: {
          inheritAttrs: false,
          template: '<slot name="panel" :option="{ value: \'master-notice\' }" />',
        },
      },
    },
  });
  await vi.waitFor(() =>
    expect(wrapper!.findComponent(BaseSettingFieldRenderer).exists()).toBe(true)
  );
  const field = wrapper.findComponent(BaseSettingFieldRenderer);
  field.vm.$emit('updateField', 'uiPassword', 'new-password');
  field.vm.$emit('updateField', 'trayTooltip', '已提交提示');
  await nextTick();
  const store = useUnsavedChangesStore(pinia);
  const saved = store.runPendingAction('base-setting');
  await vi.waitFor(() => expect(api.salt).toHaveBeenCalledOnce());
  const savingWhileWaiting = store.activePendingActions[0]?.saving;
  if (edits) {
    field.vm.$emit('updateField', 'trayTooltip', '等待期间的新提示');
    await nextTick();
  }
  releaseSalt({ data: { item: { salt: 'settings-salt' } } });
  await saved;
  expect(remote.trayTooltip).toBe('已提交提示');
  expect(api.save).toHaveBeenCalledOnce();
  expect(store.hasUnsavedChanges).toBe(edits);
  expect(field.props('model').trayTooltip).toBe(edits ? '等待期间的新提示' : '已提交提示');
  expect(savingWhileWaiting).toBe(true);
  expect(api.read).toHaveBeenCalledTimes(2); // 首屏 + 保存后刷新一次。
});

it('关闭 Markdown 时同步关闭回复名片草稿', async () => {
  const remote = {
    uiPassword: '------',
    officialQQUseMarkdown: true,
    officialQQUseNameCard: true,
    commandPrefix: ['.'],
    diceMasters: [],
    noticeIds: [],
    extDefaultSettings: [],
  };
  api.read.mockResolvedValue({ item: structuredClone(remote) });
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('settings-test');
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(BaseSettingPage, {
    global: {
      plugins: [pinia, [VueQueryPlugin, { queryClient: client }]],
      renderStubDefaultSlot: true,
      stubs: {
        ResponsiveTabs: {
          inheritAttrs: false,
          template: '<slot name="panel" :option="{ value: \'master-notice\' }" />',
        },
      },
    },
  });
  await vi.waitFor(() =>
    expect(wrapper!.findComponent(BaseSettingFieldRenderer).exists()).toBe(true)
  );
  const field = wrapper.findComponent(BaseSettingFieldRenderer);
  field.vm.$emit('updateField', 'officialQQUseMarkdown', false);
  await nextTick();

  expect(field.props('model').officialQQUseMarkdown).toBe(false);
  expect(field.props('model').officialQQUseNameCard).toBe(false);
});
