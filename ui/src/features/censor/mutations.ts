import { useMutation, type QueryClient } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import {
  deleteSdApiV2CensorFiles,
  postSdApiV2CensorConfig,
  postSdApiV2CensorFilesUpload,
  postSdApiV2CensorRestart,
  postSdApiV2CensorStop,
  type CensorConfigBody,
} from '@/api';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
} from '@/features/testMode/state';
import { invalidateCensorQueries } from './queries';

export function useCensorMutations(options: {
  queryClient: QueryClient;
  message: MessageApi;
  getConfigPayload: () => CensorConfigBody;
  onReloaded: () => void;
  onStopped: () => void;
  onConfigSaved: () => void;
  onFilesChanged: () => void;
}) {
  // 启用与重载走同一个接口，反馈文案需按调用意图区分，
  // 否则首次开启拦截会提示「重载成功」。
  const restartMutation = useMutation({
    mutationFn: async (intent: 'enable' | 'reload' = 'reload') => {
      const { data } = await postSdApiV2CensorRestart({ throwOnError: true });
      return { item: data.item, intent };
    },
    onSuccess: async ({ item, intent }) => {
      if (isTestModeResponse(item)) {
        options.message.warning('展示模式无法启用拦截');
        return;
      }
      options.message.success(intent === 'enable' ? '拦截已启用' : '重载拦截成功');
      options.onReloaded();
      await invalidateCensorQueries(options.queryClient);
    },
    onError: error => {
      if (isTestModeApiError(error)) {
        options.message.warning(getTestModeBlockMessage(error));
        return;
      }
      options.message.error('重载拦截失败');
    },
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2CensorStop({ throwOnError: true });
      return data.item;
    },
    onSuccess: async () => {
      options.message.success('拦截已关闭');
      options.onStopped();
      await invalidateCensorQueries(options.queryClient);
    },
    onError: () => {
      options.message.error('关闭拦截失败');
    },
  });

  const saveConfigMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2CensorConfig({
        body: options.getConfigPayload(),
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async () => {
      options.message.success('保存设置成功');
      options.onConfigSaved();
      await options.queryClient.invalidateQueries({ queryKey: ['censor-config'] });
    },
    onError: () => {
      options.message.error('保存设置失败');
    },
  });

  const uploadFileMutation = useMutation({
    mutationFn: async (file: File) => {
      const { data } = await postSdApiV2CensorFilesUpload({
        body: { file },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async () => {
      options.message.success('上传完成，请在全部操作完成后，手动重载拦截');
      options.onFilesChanged();
      await options.queryClient.invalidateQueries({ queryKey: ['censor-files'] });
    },
    onError: () => {
      options.message.error('上传失败');
    },
  });

  const deleteFilesMutation = useMutation({
    mutationFn: async (keys: string[]) => {
      const { data } = await deleteSdApiV2CensorFiles({
        body: { keys },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async () => {
      options.message.success('删除词库完成，请在全部操作完成后，手动重载拦截');
      options.onFilesChanged();
      await options.queryClient.invalidateQueries({ queryKey: ['censor-files'] });
    },
    onError: () => {
      options.message.error('删除词库失败');
    },
  });

  return {
    restartMutation,
    stopMutation,
    saveConfigMutation,
    uploadFileMutation,
    deleteFilesMutation,
  };
}
