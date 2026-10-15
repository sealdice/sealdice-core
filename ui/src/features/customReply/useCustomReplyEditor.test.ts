// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest';
import { defineComponent, nextTick } from 'vue';
import { mount, type VueWrapper } from '@vue/test-utils';
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { useCustomReplyEditor } from './useCustomReplyEditor';

const backend = vi.hoisted(() => ({
  desc: 'old',
  release: undefined as (() => void) | undefined,
}));
vi.mock('naive-ui', () => ({
  useMessage: () => ({ success: vi.fn(), error: vi.fn() }),
  useDialog: () => ({}),
}));
vi.mock('@/features/upload/store', () => ({ useUploadStore: () => ({ enqueue: vi.fn() }) }));
vi.mock('@/features/auth/state', async () => ({ hasAccessToken: (await import('vue')).ref(true) }));
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
afterEach(() => {
  wrapper?.unmount();
  client?.clear();
});

it.each([false, true])(
  'preserves the submitted baseline when edits during save = %s',
  async editDuringSave => {
    backend.desc = 'old';
    backend.release = undefined;
    let editor!: ReturnType<typeof useCustomReplyEditor>;
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    wrapper = mount(
      defineComponent({
        setup() {
          editor = useCustomReplyEditor();
          return () => null;
        },
      }),
      { global: { plugins: [[VueQueryPlugin, { queryClient: client }]] } }
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
