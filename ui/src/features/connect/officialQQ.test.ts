import {
  buildOfficialQQCreateConfig,
  buildOfficialQQDuplicateMessage,
  validateOfficialQQManualConfig,
} from './officialQQ.js';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
      throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
    }
  };

  assertDeepEqual(buildOfficialQQCreateConfig({
    appID: '10001',
    appSecret: 'secret',
    useWebhook: true,
  }, 'qrcode'), {
    appID: '',
    appSecret: '',
    useWebhook: true,
  });

  assertDeepEqual(validateOfficialQQManualConfig({
    appID: '',
    appSecret: 'secret',
  }), {
    valid: false,
    message: '请填写 AppID 与 AppSecret，或切换到扫码登录。',
  });

  assertDeepEqual(validateOfficialQQManualConfig({
    appID: '10001',
    appSecret: 'secret',
  }), {
    valid: true,
    message: '',
  });

  assertEqual(buildOfficialQQDuplicateMessage('OpenQQ:202401'), '该 QQ 官方机器人账号已存在：OpenQQ:202401');
});
