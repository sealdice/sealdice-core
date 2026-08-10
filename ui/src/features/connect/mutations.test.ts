import { it } from 'vitest';
import { prepareConnectCreatePayload } from './mutations';

it('skips official QQ preflight for QR mode and clears credentials', async () => {
  let preflightCalls = 0;
  const payload = await prepareConnectCreatePayload(
    {
      platform: 'officialqq',
      config: { appID: '10001', appSecret: 'secret' },
      officialQQMode: 'qrcode',
    },
    async () => {
      preflightCalls += 1;
      return { exists: false, userId: '' };
    },
  );
  if (preflightCalls !== 0) throw new Error(`unexpected preflight calls = ${preflightCalls}`);
  if (JSON.stringify(payload) !== JSON.stringify({
    platform: 'officialqq',
    config: { appID: '', appSecret: '' },
  })) {
    throw new Error(`unexpected payload = ${JSON.stringify(payload)}`);
  }
});

it('runs official QQ preflight for manual mode', async () => {
  let received: Record<string, unknown> | null = null;
  await prepareConnectCreatePayload(
    {
      platform: 'officialqq',
      config: { appID: '10001', appSecret: 'secret' },
      officialQQMode: 'manual',
    },
    async config => {
      received = config;
      return { exists: false, userId: '' };
    },
  );
  if (JSON.stringify(received) !== JSON.stringify({ appID: '10001', appSecret: 'secret' })) {
    throw new Error(`unexpected preflight config = ${JSON.stringify(received)}`);
  }
});
