import { buildOfficialQQCreateConfig, validateOfficialQQConfig } from './officialQQ.js';
import { expect, it } from 'vitest';

it('官方 QQ 扫码配置清除手动凭据，手动配置校验必填项', async () => {
  expect(
    buildOfficialQQCreateConfig(
      {
        appID: '10001',
        appSecret: 'secret',
        useWebhook: true,
      },
      'qrcode'
    )
  ).toEqual({
    appID: '',
    appSecret: '',
    useWebhook: true,
  });

  expect(
    buildOfficialQQCreateConfig(
      {
        appID: '10001',
        appSecret: 'secret',
        useWebhook: true,
        webhookPath: '/qq/webhook',
        webhookPort: 8099,
      },
      'qrcode'
    )
  ).toEqual({
    appID: '',
    appSecret: '',
    useWebhook: true,
    webhookPath: '/qq/webhook',
    webhookPort: 8099,
  });

  expect(
    validateOfficialQQConfig(
      {
        appID: '',
        appSecret: 'secret',
      },
      'manual'
    )
  ).toEqual({
    valid: false,
    message: '请填写机器人 ID 和机器人密钥，或切换到扫码登录。',
  });

  expect(
    validateOfficialQQConfig(
      {
        appID: '10001',
        appSecret: 'secret',
      },
      'manual'
    )
  ).toEqual({
    valid: true,
    message: '',
  });
});
