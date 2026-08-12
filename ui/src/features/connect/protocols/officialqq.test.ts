import { it } from 'vitest';
import { getConnectProtocolModule } from './index';

it('validates and prepares official QQ creation through the protocol module', async () => {
  const module = getConnectProtocolModule('officialqq');
  const invalid = module.validateCreate?.(
    { appID: '', appSecret: '' },
    { officialQQMode: 'manual' }
  );
  if (!invalid || invalid.valid || !invalid.message.includes('机器人 ID')) {
    throw new Error(`unexpected invalid result = ${JSON.stringify(invalid)}`);
  }

  const prepared = module.prepareCreateConfig?.(
    { appID: '10001', appSecret: 'secret', useWebhook: true },
    { officialQQMode: 'qrcode' }
  );
  if (JSON.stringify(prepared) !== JSON.stringify({ appID: '', appSecret: '', useWebhook: true })) {
    throw new Error(`unexpected prepared config = ${JSON.stringify(prepared)}`);
  }

  const qrWebhook = module.prepareCreateConfig?.(
    { appID: '10001', appSecret: 'secret', useWebhook: true, webhookPath: '/webhook', webhookPort: 8099 },
    { officialQQMode: 'qrcode' }
  );
  if (JSON.stringify(qrWebhook) !== JSON.stringify({
    appID: '',
    appSecret: '',
    useWebhook: true,
    webhookPath: '/webhook',
    webhookPort: 8099,
  })) {
    throw new Error(`unexpected QR webhook config = ${JSON.stringify(qrWebhook)}`);
  }
});

it('runs official QQ duplicate checking through the protocol module', async () => {
  const module = getConnectProtocolModule('officialqq');
  let testedConfig: Record<string, unknown> | null = null;
  await module.beforeCreate?.(
    { appID: '10001', appSecret: 'secret' },
    {
      officialQQMode: 'manual',
      testOfficialQQ: async config => {
        testedConfig = config;
        return { exists: false, userId: '' };
      },
    }
  );
  if (JSON.stringify(testedConfig) !== JSON.stringify({ appID: '10001', appSecret: 'secret' })) {
    throw new Error(`unexpected test config = ${JSON.stringify(testedConfig)}`);
  }
});
