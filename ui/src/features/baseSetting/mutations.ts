import { useMutation, type QueryClient } from '@tanstack/vue-query';
import type { MessageApi } from 'naive-ui';
import {
  getSdApiV2BaseLoginSalt,
  postSdApiV2BaseSettingMailTest,
  postSdApiV2BaseSettingUpgrade,
  putSdApiV2BaseSettingValue,
} from '@/api';
import { passwordHash } from '@/features/auth/crypto';
import type { BaseSettingValueModel } from './viewModel';

const uiPasswordMasked = '------';
const mailPasswordMasked = '******';

/**
 * 后端 SetValue 的 Body 是 map[string]any，huma 直接把请求体解成这个 map，
 * 补丁必须是请求体本身。多包一层 body 会让 applyPatch 读不到任何键 ——
 * 请求照样返回成功，但设置没有变化。
 */
export async function saveBaseSettingValue(payload: Record<string, unknown>) {
  const { data } = await putSdApiV2BaseSettingValue({
    body: payload,
    throwOnError: true,
  });
  return data.item;
}

export function useBaseSettingMutations(options: {
  queryClient: QueryClient;
  message: MessageApi;
  onSaved: () => void;
}) {
  const saveMutation = useMutation({
    mutationFn: saveBaseSettingValue,
    onSuccess: async () => {
      options.message.success('保存设置成功');
      options.onSaved();
      await options.queryClient.invalidateQueries({ queryKey: ['base-setting-value'] });
    },
    onError: () => {
      options.message.error('保存设置失败');
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
    onError: () => {
      options.message.error('发送测试邮件失败');
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
    onError: () => {
      options.message.error('上传固件包失败');
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
  ) => Record<string, unknown>
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
