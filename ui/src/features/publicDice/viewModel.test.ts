import {
  buildPublicDicePayload,
  createPublicDiceDraft,
  getPublicDiceEndpointRows,
  isPublicDiceDirty,
  normalizePublicDiceConfig,
} from './viewModel.js';
import {
  getEndpointProtocolLabel,
  getEndpointStateMeta,
} from '@/features/connect/endpointDisplay.js';
import { expect, it } from 'vitest';

it('公开骰子配置与账号选择转换', async () => {
  expect(
    normalizePublicDiceConfig({
      publicDiceEnable: true,
      publicDiceId: undefined as unknown as string,
      publicDiceName: ' 公骰 ',
      publicDiceAvatar: null as unknown as string,
      publicDiceNote: ' 留言 ',
      publicDiceBrief: ' 简介 ',
    })
  ).toEqual({
    publicDiceEnable: true,
    publicDiceId: '',
    publicDiceName: '公骰',
    publicDiceAvatar: '',
    publicDiceNote: '留言',
    publicDiceBrief: '简介',
  });

  expect(
    createPublicDiceDraft({
      config: {
        publicDiceEnable: true,
        publicDiceId: 'dice-id',
        publicDiceName: '公骰',
        publicDiceAvatar: '',
        publicDiceNote: '',
        publicDiceBrief: '',
      },
      endpoints: [
        {
          id: 'ep-1',
          userId: '1001',
          platform: 'QQ',
          protocolType: 'onebot',
          state: 1,
          isPublic: true,
        },
        {
          id: 'ep-2',
          userId: '1002',
          platform: 'QQ',
          protocolType: 'milky',
          state: 0,
          isPublic: false,
        },
      ],
    })
  ).toEqual({
    config: {
      publicDiceEnable: true,
      publicDiceId: 'dice-id',
      publicDiceName: '公骰',
      publicDiceAvatar: '',
      publicDiceNote: '',
      publicDiceBrief: '',
    },
    selectedEndpointIds: ['ep-1'],
  });

  expect(
    buildPublicDicePayload(
      {
        publicDiceEnable: true,
        publicDiceId: ' dice-id ',
        publicDiceName: ' 公骰 ',
        publicDiceAvatar: ' https://example.com/avatar.png ',
        publicDiceNote: ' 留言 ',
        publicDiceBrief: ' 简介 ',
      },
      [' ep-2 ', 'ep-1', 'ep-1', '']
    )
  ).toEqual({
    config: {
      publicDiceEnable: true,
      publicDiceId: 'dice-id',
      publicDiceName: '公骰',
      publicDiceAvatar: 'https://example.com/avatar.png',
      publicDiceNote: '留言',
      publicDiceBrief: '简介',
    },
    selectedEndpointIds: ['ep-1', 'ep-2'],
  });

  const baseDraft = {
    config: {
      publicDiceEnable: true,
      publicDiceId: 'dice-id',
      publicDiceName: '公骰',
      publicDiceAvatar: '',
      publicDiceNote: '',
      publicDiceBrief: '',
    },
    selectedEndpointIds: ['ep-1', 'ep-2'],
  };
  expect(
    isPublicDiceDirty(baseDraft, { ...baseDraft, selectedEndpointIds: ['ep-2', 'ep-1'] })
  ).toBe(false);
  expect(isPublicDiceDirty(baseDraft, { ...baseDraft, selectedEndpointIds: ['ep-1'] })).toBe(true);

  expect(getEndpointStateMeta(1)).toEqual({ text: '已连接', tagType: 'success' });
  expect(getEndpointStateMeta(2)).toEqual({ text: '连接中', tagType: 'warning' });
  expect(getEndpointStateMeta(3)).toEqual({ text: '失败', tagType: 'error' });
  expect(getEndpointStateMeta(0)).toEqual({ text: '断开', tagType: 'error' });

  expect(
    getEndpointProtocolLabel({
      platform: 'QQ',
      protocolType: 'onebot',
      adapter: { builtinMode: 'lagrange' },
    })
  ).toBe('QQ（内置客户端）');
  expect(
    getEndpointProtocolLabel({
      platform: 'QQ',
      protocolType: 'milky',
      adapter: { built_in_mode: 'milky' },
    })
  ).toBe('QQ（内置 Milky）');
  expect(
    getEndpointProtocolLabel({
      platform: 'QQ',
      protocolType: 'milky',
    })
  ).toBe('QQ（Milky）');
  expect(
    getEndpointProtocolLabel({
      platform: 'Discord',
      protocolType: 'satori',
    })
  ).toBe('Satori');

  expect(
    getPublicDiceEndpointRows([
      {
        id: 'ep-1',
        userId: '1001',
        platform: 'QQ',
        protocolType: 'onebot',
        state: 1,
        isPublic: true,
      },
      {
        id: 'ep-2',
        userId: '1002',
        platform: 'QQ',
        protocolType: 'milky',
        state: 3,
        isPublic: false,
      },
    ])
  ).toEqual([
    {
      id: 'ep-1',
      userId: '1001',
      platform: 'QQ',
      protocol: 'QQ',
      protocolType: 'onebot',
      stateText: '已连接',
      stateTagType: 'success',
      isPublic: true,
    },
    {
      id: 'ep-2',
      userId: '1002',
      platform: 'QQ',
      protocol: 'QQ（Milky）',
      protocolType: 'milky',
      stateText: '失败',
      stateTagType: 'error',
      isPublic: false,
    },
  ]);
});
