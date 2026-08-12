export type OfficialQQMode = 'manual' | 'qrcode';

export function buildOfficialQQCreateConfig(config: Record<string, unknown>, mode: OfficialQQMode) {
  const { webhookPath, webhookPort, ...credentials } = config;
  const next = {
    ...credentials,
    ...(mode === 'qrcode' ? { appID: '', appSecret: '' } : {}),
    useWebhook: Boolean(config.useWebhook),
  };
  if (!next.useWebhook) return next;
  return { ...next, webhookPath, webhookPort };
}

export function validateOfficialQQManualConfig(config: Record<string, unknown>) {
  return validateOfficialQQConfig(config, 'manual');
}

export function validateOfficialQQConfig(config: Record<string, unknown>, mode: OfficialQQMode) {
  if (config.useWebhook) {
    const webhookPath = String(config.webhookPath ?? '').trim();
    const webhookPort = Number(config.webhookPort);
    if (
      !webhookPath.startsWith('/') ||
      !Number.isInteger(webhookPort) ||
      webhookPort < 1 ||
      webhookPort > 65535
    ) {
      return {
        valid: false,
        message: '请填写以 / 开头的 Webhook 路径和 1-65535 范围内的端口。',
      };
    }
  }

  if (mode === 'manual') {
    const appID = String(config.appID ?? '').trim();
    const appSecret = String(config.appSecret ?? '').trim();
    if (!appID || !appSecret) {
      return {
        valid: false,
        message: '请填写机器人 ID 和机器人密钥，或切换到扫码登录。',
      };
    }
  }
  return {
    valid: true,
    message: '',
  };
}

export function buildOfficialQQDuplicateMessage(userId: string) {
  return `该 QQ 官方机器人账号已存在：${userId}`;
}
