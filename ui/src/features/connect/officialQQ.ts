export type OfficialQQMode = 'manual' | 'qrcode';

export function buildOfficialQQCreateConfig(config: Record<string, unknown>, mode: OfficialQQMode) {
  if (mode !== 'qrcode') return { ...config };
  return {
    ...config,
    appID: '',
    appSecret: '',
  };
}

export function validateOfficialQQManualConfig(config: Record<string, unknown>) {
  const appID = String(config.appID ?? '').trim();
  const appSecret = String(config.appSecret ?? '').trim();
  if (!appID || !appSecret) {
    return {
      valid: false,
      message: '请填写 AppID 与 AppSecret，或切换到扫码登录。',
    };
  }
  return {
    valid: true,
    message: '',
  };
}

export function buildOfficialQQDuplicateMessage(userId: string) {
  return `该 QQ 官方机器人账号已存在：${userId}`;
}
