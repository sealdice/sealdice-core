// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';
import { mount, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { useCustomReplyEditor } from './useCustomReplyEditor';
import { createPinia, disposePinia, type Pinia } from 'pinia';
import { useAuthStore } from '@/features/auth/store';
import { VueDraggableNext } from 'vue-draggable-next';
import NestedRuleEditor from '@/components/custom-reply/NestedRuleEditor.vue';
import ConditionBuilder from '@/components/custom-reply/ConditionBuilder.vue';
import type { ReplyTask } from './model';

const backend = vi.hoisted(() => ({
  desc: 'old',
  release: undefined as (() => void) | undefined,
}));
vi.mock('naive-ui', async original => ({
  ...(await original<typeof import('naive-ui')>()),
  useMessage: () => ({ success: vi.fn(), error: vi.fn() }),
  useDialog: () => ({}),
}));
vi.mock('@/features/upload/store', () => ({ useUploadStore: () => ({ enqueue: vi.fn() }) }));
vi.mock('@/features/unsavedChanges', () => ({ useUnsavedChanges: vi.fn() }));
vi.mock('@/api/download', () => ({ downloadApiFile: vi.fn() }));
vi.mock('@/api', () => ({
  getSdApiV2ConfigReplyOptions: () => ({
    queryKey: ['config'],
    queryFn: async () => ({ item: { customReplyConfigEnable: true } }),
  }),
  getSdApiV2ConfigReplyQueryKey: () => ['config'],
  getSdApiV2CustomReplyFiles: async () => ({
    data: { item: { list: [{ filename: 'reply.yaml' }], total: 1 } },
  }),
  getSdApiV2CustomReplyFilesByFilename: async () => ({
    data: {
      item: {
        filename: 'reply.yaml',
        enable: true,
        vmVersion: 'v2',
        interval: 0,
        name: 'test',
        author: [],
        version: '1',
        createTimestamp: 1,
        updateTimestamp: 1,
        desc: backend.desc,
        storeID: '',
        conditions: [],
        itemCount: 0,
      },
    },
  }),
  getSdApiV2CustomReplyFilesByFilenameRules: async () => ({
    data: { item: { list: [], total: 0 } },
  }),
  putSdApiV2CustomReplyFilesByFilename: ({ body }: { body: { desc: string } }) =>
    new Promise(resolve => {
      backend.release = () => {
        backend.desc = body.desc;
        resolve({ data: { item: { success: true } } });
      };
    }),
  deleteSdApiV2CustomReplyFilesByFilename: vi.fn(),
  getSdApiV2CustomReplyFilesByFilenameDownload: vi.fn(),
  postSdApiV2CustomReplyFiles: vi.fn(),
  putSdApiV2ConfigReply: vi.fn(),
}));

let wrapper: VueWrapper | undefined;
let client: QueryClient;
let pinia: Pinia;
afterEach(() => {
  wrapper?.unmount();
  client?.clear();
  if (pinia) disposePinia(pinia);
});

it('条件和规则输入只发编辑事件，排序不直接修改传入数组', async () => {
  const tasks: ReplyTask[] = [
    {
      enable: true,
      conditions: [{ condType: 'textMatch', matchType: 'matchExact', value: 'before' }],
      results: [{ resultType: 'replyToSender', delay: 0, message: [['reply', 2]] }],
    },
  ];
  const before = structuredClone(tasks);
  wrapper = mount(NestedRuleEditor, { props: { tasks } });
  await wrapper.get('.condition-value input').setValue('edited condition');
  await wrapper.get('.reply-text textarea').setValue('edited reply');
  expect(wrapper.emitted('updateCondition')?.at(-1)).toEqual([0, 0, { value: 'edited condition' }]);
  expect(wrapper.emitted('updateResult')?.at(-1)).toEqual([
    0,
    0,
    { message: [['edited reply', 2]] },
  ]);
  wrapper.getComponent(VueDraggableNext).vm.$emit('update:modelValue', [...tasks]);
  expect(wrapper.emitted('reorder')?.at(-1)).toEqual([tasks]);
  expect(tasks).toEqual(before);
});

it('分页只派生视图，编辑和排序写入整份草稿且不会影响其他页', async () => {
  let editor!: ReturnType<typeof useCustomReplyEditor>;
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  pinia = createPinia();
  useAuthStore(pinia).setAccessToken('test');
  wrapper = mount(
    defineComponent({
      setup() {
        editor = useCustomReplyEditor();
        return () =>
          h('div', [
            h(ConditionBuilder, {
              conditions: editor.pagedCommonConditions.value,
              onUpdateCondition: editor.updateCommonCondition,
            }),
            h(NestedRuleEditor, {
              tasks: editor.rulePageItems.value,
              startIndex: editor.rulePageStart.value,
              onReorder: editor.reorderRules,
              onUpdateCondition: editor.updateRuleCondition,
              onUpdateResult: editor.updateRuleResult,
            }),
          ]);
      },
    }),
    { global: { plugins: [pinia, [VueQueryPlugin, { queryClient: client }]] } }
  );
  await vi.waitFor(() => expect(editor.currentFileDraft.value).not.toBeNull());
  editor.rulesPageSize.value = 2;
  editor.commonConditionsPageSize.value = 1;
  editor.addCommonCondition();
  editor.addCommonCondition();
  for (let i = 0; i < 4; i++) editor.addReplyItem();
  await nextTick();
  const draft = editor.currentFileDraft.value!;
  const firstPage = draft.items.slice(0, 2);
  const secondPage = draft.items.slice(2);
  await wrapper
    .getComponent(ConditionBuilder)
    .get('.condition-value input')
    .setValue('common page 2');
  const rules = wrapper.getComponent(NestedRuleEditor);
  await rules.findAll('.condition-value input')[0]!.setValue('rule page 2');
  await rules.findAll('.reply-text textarea')[0]!.setValue('saved reply');
  rules.getComponent(VueDraggableNext).vm.$emit('update:modelValue', [...secondPage].reverse());
  await nextTick();
  expect(draft.conditions.map(item => item.value)).toEqual(['要匹配的文本', 'common page 2']);
  expect(draft.items.slice(0, 2)).toEqual(firstPage);
  expect(draft.items[3]).toBe(secondPage[0]);
  expect(draft.items[3]!.conditions[0]!.value).toBe('rule page 2');
  expect(draft.items[3]!.results[0]!.message[0]![0]).toBe('saved reply');
  editor.rulesPage.value = 1;
  await nextTick();
  editor.rulesPage.value = 2;
  await nextTick();
  expect(editor.rulePageItems.value).toEqual([...secondPage].reverse());
  expect(editor.modified.value).toBe(true);
  editor.selectFile('another.yaml');
  expect(editor.fileSwitchDialogVisible.value).toBe(true);
});

it.each([false, true])(
  'preserves the submitted baseline when edits during save = %s',
  async editDuringSave => {
    backend.desc = 'old';
    backend.release = undefined;
    let editor!: ReturnType<typeof useCustomReplyEditor>;
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    pinia = createPinia();
    useAuthStore(pinia).setAccessToken('test');
    wrapper = mount(
      defineComponent({
        setup() {
          editor = useCustomReplyEditor();
          return () => null;
        },
      }),
      { global: { plugins: [pinia, [VueQueryPlugin, { queryClient: client }]] } }
    );
    await vi.waitFor(() => expect(editor.currentFileDraft.value).not.toBeNull());
    await nextTick();
    editor.currentFileDraft.value!.desc = 'submitted A';
    await nextTick();
    const pending = editor.saveCurrent();
    await vi.waitFor(() => expect(backend.release).toBeTypeOf('function'));
    if (editDuringSave) editor.currentFileDraft.value!.desc = 'new unsaved B';
    await nextTick();
    backend.release!();
    await pending;
    await nextTick();
    expect(backend.desc).toBe('submitted A');
    expect(editor.currentFileDraft.value!.desc).toBe(
      editDuringSave ? 'new unsaved B' : 'submitted A'
    );
    expect(editor.modified.value).toBe(editDuringSave);
  }
);
