import { useMutation } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import type { BaseSettingPatch } from '@/api';
import {
  getSdApiV2BaseLoginSalt,
  postSdApiV2BaseSettingMailTest,
  postSdApiV2BaseSettingUpgrade,
  putSdApiV2BaseSettingValue,
} from '@/api';
import { passwordHash } from '@/features/auth/crypto';
import { notifyError } from '@/features/feedback/notifyError';

const uiPasswordMasked = '------';
const mailPasswordMasked = '******';

export async function saveBaseSettingValue(payload: BaseSettingPatch) {
  const body = { ...payload };
  if (
    typeof body.uiPassword === 'string' &&
    body.uiPassword !== '' &&
    body.uiPassword !== uiPasswordMasked
  ) {
    const salt = await getSdApiV2BaseLoginSalt({ throwOnError: true });
    body.uiPassword = await passwordHash(salt.data.item.salt, body.uiPassword);
  } else {
    delete body.uiPassword;
  }
  if (body.mailPassword === mailPasswordMasked) delete body.mailPassword;

  const { data } = await putSdApiV2BaseSettingValue({
    body,
    throwOnError: true,
  });
  return data.item;
}

export function useBaseSettingMutations(message: MessageApi) {
  const saveMutation = useMutation({
    mutationFn: saveBaseSettingValue,
    onSuccess: () => {
      message.success('保存设置成功');
    },
    onError: error => {
      notifyError(error, message, '保存设置失败');
    },
  });

  const mailTestMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2BaseSettingMailTest({ throwOnError: true });
      return data.item;
    },
    onSuccess: () => {
      message.success('已尝试发送测试邮件');
    },
    onError: error => {
      notifyError(error, message, '发送测试邮件失败');
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
      message.info('开始上传固件包，更新时程序可能离线');
    },
    onError: error => {
      notifyError(error, message, '上传固件包失败');
    },
  });

  return {
    saveMutation,
    mailTestMutation,
    upgradeMutation,
  };
}
