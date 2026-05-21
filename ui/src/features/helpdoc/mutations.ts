import { useMutation, type QueryClient } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import {
  postSdApiV2HelpdocConfig,
  postSdApiV2HelpdocDelete,
  postSdApiV2HelpdocReload,
} from '@/api';
import { invalidateHelpdocQueries } from './queries';

export function useHelpdocMutations(options: {
  queryClient: QueryClient;
  message: MessageApi;
  getConfigPayload: () => Record<string, string[]>;
  onReloaded: () => void;
  onDeleted: () => void;
  onConfigSaved: () => void;
}) {
  const reloadMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2HelpdocReload({ throwOnError: true });
      return data.item;
    },
    onSuccess: async item => {
      if (item.testMode) {
        options.message.warning('展示模式无法重载帮助文档');
        return;
      }
      options.message.success('重载帮助文件成功');
      options.onReloaded();
      await invalidateHelpdocQueries(options.queryClient);
    },
    onError: () => {
      options.message.error('重载帮助文件失败');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (keys: string[]) => {
      const { data } = await postSdApiV2HelpdocDelete({
        body: { keys },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async item => {
      if (!item.success) {
        options.message.error(item.err || '删除文件失败');
        return;
      }
      options.message.success('删除文件成功');
      options.onDeleted();
      await options.queryClient.invalidateQueries({ queryKey: ['helpdoc-tree'] });
    },
    onError: () => {
      options.message.error('删除文件失败');
    },
  });

  const saveConfigMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2HelpdocConfig({
        body: {
          aliases: options.getConfigPayload(),
        },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async item => {
      if (!item.success) {
        options.message.error(item.err || '保存设置失败');
        return;
      }
      options.message.success('保存设置成功');
      options.onConfigSaved();
      await options.queryClient.invalidateQueries({ queryKey: ['helpdoc-config'] });
    },
    onError: () => {
      options.message.error('保存设置失败');
    },
  });

  return {
    reloadMutation,
    deleteMutation,
    saveConfigMutation,
  };
}
