import { useMutation } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import {
  deleteSdApiV2ImconnectionById,
  postSdApiV2Imconnection,
  postSdApiV2ImconnectionOfficialqqTest,
  putSdApiV2ImconnectionById,
  putSdApiV2ImconnectionByIdEnable,
  type EndPointInfo,
} from '@/api';
import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
import { getTestModeBlockMessage, isTestModeApiError } from '@/features/testMode/state';
import type { OfficialQQMode } from './officialQQ';
import { getConnectProtocolModule } from './protocols';

export type ConnectCreateInput = {
  platform: string;
  config: DynamicFormModel;
  officialQQMode: OfficialQQMode;
};

export type ConnectCreatePayload = {
  platform: string;
  config: Record<string, unknown>;
};

export type ConnectCreatedResult = {
  endpoint: EndPointInfo;
  platform: string;
  officialQQMode: OfficialQQMode;
};

type OfficialQQTest = (
  config: Record<string, unknown>
) => Promise<{ exists: boolean; userId: string }>;

export async function prepareConnectCreatePayload(
  input: ConnectCreateInput,
  testOfficialQQ: OfficialQQTest
): Promise<ConnectCreatePayload> {
  const module = getConnectProtocolModule(input.platform);
  const context = {
    officialQQMode: input.officialQQMode,
    testOfficialQQ,
  };
  const config = module.prepareCreateConfig?.({ ...input.config }, context) ?? { ...input.config };
  await module.beforeCreate?.(config, context);
  return { platform: input.platform, config };
}

export function useConnectMutations(options: {
  message: MessageApi;
  onCreated: (result: ConnectCreatedResult) => void;
  onUpdated: () => void;
  onEnabled: () => void;
  onDeleted: () => void;
}) {
  const testOfficialQQ: OfficialQQTest = async config => {
    const { data } = await postSdApiV2ImconnectionOfficialqqTest({
      body: { config },
      throwOnError: true,
    });
    return {
      exists: data.item.exists,
      userId: data.item.userId,
    };
  };

  const createMutation = useMutation({
    mutationFn: async (input: ConnectCreateInput): Promise<ConnectCreatedResult> => {
      const payload = await prepareConnectCreatePayload(input, testOfficialQQ);
      const { data } = await postSdApiV2Imconnection({
        body: payload,
        throwOnError: true,
      });
      return {
        endpoint: data.item,
        platform: input.platform,
        officialQQMode: input.officialQQMode,
      };
    },
    onSuccess: result => {
      options.message.success('账号已添加');
      options.onCreated(result);
    },
    onError: error => {
      if (isTestModeApiError(error)) {
        options.message.warning(getTestModeBlockMessage(error));
        return;
      }
      options.message.error('添加账号失败');
    },
  });

  const updateMutation = useMutation({
    mutationFn: async (input: { id: string; config: DynamicFormModel }) => {
      const { data } = await putSdApiV2ImconnectionById({
        path: { id: input.id },
        body: input.config,
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: () => {
      options.message.success('账号配置已更新');
      options.onUpdated();
    },
    onError: error => {
      if (isTestModeApiError(error)) {
        options.message.warning(getTestModeBlockMessage(error));
        return;
      }
      options.message.error('账号配置更新失败');
    },
  });

  const enableMutation = useMutation({
    mutationFn: async (input: { id: string; enable: boolean }) => {
      const { data } = await putSdApiV2ImconnectionByIdEnable({
        path: { id: input.id },
        body: { enable: input.enable },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: () => {
      options.message.success('账号状态已更新');
      options.onEnabled();
    },
    onError: error => {
      if (isTestModeApiError(error)) {
        options.message.warning(getTestModeBlockMessage(error));
        return;
      }
      options.message.error('账号状态更新失败');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const { data } = await deleteSdApiV2ImconnectionById({
        path: { id },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: () => {
      options.message.success('账号已删除');
      options.onDeleted();
    },
    onError: error => {
      if (isTestModeApiError(error)) {
        options.message.warning(getTestModeBlockMessage(error));
        return;
      }
      options.message.error('删除账号失败');
    },
  });

  return { createMutation, updateMutation, enableMutation, deleteMutation };
}
