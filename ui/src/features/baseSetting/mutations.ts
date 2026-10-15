import { useMutation, type QueryClient } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import type { BaseSettingPatch } from '@/api';
import {
  getSdApiV2BaseLoginSalt,
  getSdApiV2BaseSettingValueQueryKey,
  postSdApiV2BaseSettingMailTest,
  postSdApiV2BaseSettingUpgrade,
  putSdApiV2BaseSettingValue,
} from '@/api';
import { passwordHash } from '@/features/auth/crypto';
import type { BaseSettingValueModel } from './viewModel';
import { notifyError } from '@/features/feedback/notifyError';

const uiPasswordMasked = '------';
const mailPasswordMasked = '******';

export async function saveBaseSettingValue(payload: BaseSettingPatch) {
  const { data } = await putSdApiV2BaseSettingValue({
    body: payload,
    throwOnError: true,
  });
  return data.item;
}

export function useBaseSettingMutations(options: {
  queryClient: QueryClient;
  message: MessageApi;
}) {
  const saveMutation = useMutation({
    mutationFn: saveBaseSettingValue,
    onSuccess: async () => {
      options.message.success('保存设置成功');
      await options.queryClient.invalidateQueries({
        queryKey: getSdApiV2BaseSettingValueQueryKey(),
      });
    },
    onError: error => {
      notifyError(error, options.message, '保存设置失败');
    },
  });

  const mailTestMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2BaseSettingMailTest({ throwOnError: true });
      return data.item;
    },
    onSuccess: () => {
      options.message.success('已尝试发送测试邮件');
    },
    onError: error => {
      notifyError(error, options.message, '发送测试邮件失败');
    },
  });

  const upgradeMutation = useMutation({
    mutationFn: async (file: File) => {
      const { data } = await postSdApiV2BaseSettingUpgrade({
        body: { file },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: () => {
      options.message.info('开始上传固件包，更新时程序可能离线');
    },
    onError: error => {
      notifyError(error, options.message, '上传固件包失败');
    },
  });

  return {
    saveMutation,
    mailTestMutation,
    upgradeMutation,
  };
}

export async function prepareBaseSettingSavePayload(
  current: BaseSettingValueModel,
  initial: BaseSettingValueModel,
  diffBuilder: (
    currentValue: BaseSettingValueModel,
    initialValue: BaseSettingValueModel
  ) => BaseSettingPatch
) {
  const payload = diffBuilder(current, initial);
  if (
    typeof payload.uiPassword === 'string' &&
    payload.uiPassword !== '' &&
    payload.uiPassword !== uiPasswordMasked
  ) {
    const salt = await getSdApiV2BaseLoginSalt({ throwOnError: true });
    payload.uiPassword = await passwordHash(salt.data.item.salt, payload.uiPassword);
  } else {
    delete payload.uiPassword;
  }

  if (payload.mailPassword === mailPasswordMasked) {
    delete payload.mailPassword;
  }

  return payload;
}
