import {
  buildOfficialQQCreateConfig,
  buildOfficialQQDuplicateMessage,
  validateOfficialQQManualConfig,
} from './officialQQ.js';
import { expect, it } from 'vitest';

it('官方 QQ 配置与探测结果转换', async () => {
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
    validateOfficialQQManualConfig({
      appID: '',
      appSecret: 'secret',
    })
  ).toEqual({
    valid: false,
    message: '请填写机器人 ID 和机器人密钥，或切换到扫码登录。',
  });

  expect(
    validateOfficialQQManualConfig({
      appID: '10001',
      appSecret: 'secret',
    })
  ).toEqual({
    valid: true,
    message: '',
  });

  expect(buildOfficialQQDuplicateMessage('OpenQQ:202401')).toBe(
    '该 QQ 官方机器人账号已存在：OpenQQ:202401'
  );
});
