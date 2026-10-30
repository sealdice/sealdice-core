// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';
import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { NButton, NCheckbox } from 'naive-ui';
import { client } from '@/api';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import GroupPage from './group.vue';

const feedback = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn(), warning: vi.fn() }));
vi.mock('naive-ui', async original => ({
  ...(await original<typeof import('naive-ui')>()),
  useMessage: () => feedback,
}));
vi.mock('pro-naive-ui', () => ({
  createProSearchForm: () => ({}),
  ProSearchForm: { template: '<div />' },
}));

const originalAdapter = client.instance.defaults.adapter;
let wrapper: VueWrapper | undefined;
let queryClient: QueryClient;

beforeEach(() => {
  useAuthStore(appPinia).setAccessToken('group-test');
  useAuthStore(appPinia).finishInitialization();
});

afterEach(async () => {
  wrapper?.unmount();
  queryClient?.clear();
  useAuthStore(appPinia).clearAccessToken();
  await nextTick();
  client.instance.defaults.adapter = originalAdapter;
  vi.clearAllMocks();
});

async function mountPage(quitResponse: (body: Record<string, unknown>) => unknown) {
  const requests: Record<string, unknown>[] = [];
  client.instance.defaults.adapter = async config => {
    const path = new URL(config.url!, 'http://local').pathname;
    let data: unknown = { item: [] };
    if (path.endsWith('/list')) {
      data = {
        item: {
          total: 2,
          list: [2010, 2020].map(id => ({
            groupId: `QQ-Group:${id}`,
            groupName: `测试群 ${id}`,
            active: true,
            diceIdExistsMap: { 'QQ:1001': true },
          })),
        },
      };
    } else if (path.endsWith('/quit')) {
      const body = JSON.parse(config.data) as Record<string, unknown>;
      requests.push(body);
      data = quitResponse(body);
    }
    return { config, data, status: 200, statusText: 'OK', headers: {} };
  };
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  wrapper = shallowMount(GroupPage, {
    global: {
      plugins: [appPinia, [VueQueryPlugin, { queryClient }]],
      renderStubDefaultSlot: true,
      stubs: {
        NModal: { props: ['show'], template: '<div v-if="show"><slot /></div>' },
        ListPanel: { template: '<section><slot name="toolbar" /><slot /></section>' },
        ResultToolbar: { template: '<div><slot name="meta" /><slot /></div>' },
        FoldableCard: {
          template: '<section><slot name="title" /><slot name="action" /><slot /></section>',
        },
      },
    },
  });
  await flushPromises();
  return requests;
}

function button(text: string) {
  const result = wrapper!.findAllComponents(NButton).find(item => item.text().trim() === text);
  if (!result) throw new Error(`Missing button: ${text}`);
  return result;
}

async function click(text: string) {
  button(text).vm.$emit('click');
  await flushPromises();
}

async function acknowledgeOwnerQuit() {
  const checkbox = wrapper!
    .findAllComponents(NCheckbox)
    .find(item => item.text().includes('我已知晓'));
  if (!checkbox) throw new Error('Missing owner acknowledgement');
  checkbox.vm.$emit('update:checked', true);
  await nextTick();
}

const confirmation = {
  groupId: 'QQ-Group:2010',
  diceId: 'QQ:1001',
  message: '继续退出将直接解散群聊。',
  confirmCode: '4321',
};

it.each([false, true])('群主退群需展示解散警告并二次确认（批量=%s）', async batch => {
  const requests = await mountPage(body => {
    const code = batch
      ? (body.confirmCodes as Record<string, string>)[confirmation.groupId]
      : body.confirmCode;
    return batch
      ? { item: code ? 2 : 0, confirmations: code ? [] : [confirmation] }
      : code
        ? { message: 'ok' }
        : confirmation;
  });
  if (batch) {
    const selectAll = wrapper!
      .findAllComponents(NCheckbox)
      .find(item => item.attributes('aria-label') === '全选当前页群组');
    if (!selectAll) throw new Error('Missing select-all checkbox');
    selectAll.vm.$emit('update:checked', true);
    await nextTick();
  }
  await click(batch ? '批量退群' : '退出 1001');
  await click('确认退群');
  expect(requests).toHaveLength(1);
  expect(wrapper!.text()).toContain('继续退出将直接解散群聊');
  expect(wrapper!.text()).toContain(confirmation.groupId);
  expect(button('确认解散并退出').props('disabled')).toBe(true);
  await click('确认解散并退出');
  expect(requests).toHaveLength(1);
  await acknowledgeOwnerQuit();
  await click('确认解散并退出');
  expect(requests).toHaveLength(2);
  expect(requests[1]).toMatchObject(
    batch
      ? { groupIds: ['QQ-Group:2010', 'QQ-Group:2020'], confirmCodes: { 'QQ-Group:2010': '4321' } }
      : { groupId: confirmation.groupId, diceId: confirmation.diceId, confirmCode: '4321' }
  );
  expect(feedback.success).toHaveBeenCalled();
});
