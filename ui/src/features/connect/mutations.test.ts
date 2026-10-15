import { expect, it, vi } from 'vitest';
import { prepareConnectCreatePayload } from './mutations';

it('扫码模式清除手动凭据并跳过探测', async () => {
  const probe = vi.fn();
  const payload = await prepareConnectCreatePayload(
    {
      platform: 'officialqq',
      config: { appID: '10001', appSecret: 'secret' },
      officialQQMode: 'qrcode',
    },
    probe
  );
  expect(probe).not.toHaveBeenCalled();
  expect(payload.config).toEqual({ appID: '', appSecret: '', useWebhook: false });
});

it('扫码模式保留已启用的 Webhook 配置', async () => {
  const probe = vi.fn();
  const payload = await prepareConnectCreatePayload(
    {
      platform: 'officialqq',
      officialQQMode: 'qrcode',
      config: {
        appID: '10001',
        appSecret: 'secret',
        useWebhook: true,
        webhookPath: '/webhook',
        webhookPort: 8099,
      },
    },
    probe
  );
  expect(probe).not.toHaveBeenCalled();
  expect(payload.config).toEqual({
    appID: '',
    appSecret: '',
    useWebhook: true,
    webhookPath: '/webhook',
    webhookPort: 8099,
  });
});

it('手动模式验证凭据并探测重复账号', async () => {
  const probe = vi.fn().mockResolvedValue({ exists: false, userId: 'OpenQQ:123' });
  const input = {
    platform: 'officialqq',
    config: { appID: '10001', appSecret: 'secret' },
    officialQQMode: 'manual' as const,
  };
  const payload = await prepareConnectCreatePayload(input, probe);
  expect(probe).toHaveBeenCalledWith({ appID: '10001', appSecret: 'secret', useWebhook: false });
  expect(payload.config).toEqual({ appID: '10001', appSecret: 'secret', useWebhook: false });

  probe.mockResolvedValue({ exists: true, userId: 'OpenQQ:123' });
  await expect(prepareConnectCreatePayload(input, probe)).rejects.toThrow('账号已存在');
});

it('缺少手动凭据时不发送探测请求', async () => {
  const probe = vi.fn();
  await expect(
    prepareConnectCreatePayload(
      {
        platform: 'officialqq',
        config: {},
        officialQQMode: 'manual',
      },
      probe
    )
  ).rejects.toThrow('机器人 ID');
  expect(probe).not.toHaveBeenCalled();
});

it('其它协议直接提交配置，不经过官方 QQ 的特例', async () => {
  const probe = vi.fn();
  const payload = await prepareConnectCreatePayload(
    {
      platform: 'milky',
      config: { token: 'secret', wsGateway: 'ws://localhost' },
      officialQQMode: 'manual',
    },
    probe
  );
  expect(probe).not.toHaveBeenCalled();
  expect(payload).toEqual({
    platform: 'milky',
    config: { token: 'secret', wsGateway: 'ws://localhost' },
  });
});
