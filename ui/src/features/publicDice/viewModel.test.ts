import {
  buildPublicDicePayload,
  createPublicDiceDraft,
  getPublicDiceEndpointRows,
  isPublicDiceDirty,
  normalizePublicDiceConfig,
} from './viewModel.js';
import { getEndpointProtocolLabel, getEndpointStateMeta } from '@/features/connect/endpointDisplay.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  const normalize = (value: unknown): string => {
    if (Array.isArray(value)) {
      return `[${value.map(normalize).join(',')}]`;
    }
    if (value && typeof value === 'object') {
      const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b));
      return `{${entries.map(([key, entry]) => `${key}:${normalize(entry)}`).join(',')}}`;
    }
    return JSON.stringify(value);
  };
  if (normalize(actual) !== normalize(expected)) {
    throw new Error(`expected ${normalize(expected)}, got ${normalize(actual)}`);
  }
};

assertDeepEqual(normalizePublicDiceConfig({
  publicDiceEnable: true,
  publicDiceId: undefined as unknown as string,
  publicDiceName: ' 公骰 ',
  publicDiceAvatar: null as unknown as string,
  publicDiceNote: ' 留言 ',
  publicDiceBrief: ' 简介 ',
}), {
  publicDiceEnable: true,
  publicDiceId: '',
  publicDiceName: '公骰',
  publicDiceAvatar: '',
  publicDiceNote: '留言',
  publicDiceBrief: '简介',
});

assertDeepEqual(createPublicDiceDraft({
  config: {
    publicDiceEnable: true,
    publicDiceId: 'dice-id',
    publicDiceName: '公骰',
    publicDiceAvatar: '',
    publicDiceNote: '',
    publicDiceBrief: '',
  },
  endpoints: [
    { id: 'ep-1', userId: '1001', platform: 'QQ', protocolType: 'onebot', state: 1, isPublic: true },
    { id: 'ep-2', userId: '1002', platform: 'QQ', protocolType: 'milky', state: 0, isPublic: false },
  ],
}), {
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

assertDeepEqual(buildPublicDicePayload({
  publicDiceEnable: true,
  publicDiceId: ' dice-id ',
  publicDiceName: ' 公骰 ',
  publicDiceAvatar: ' https://example.com/avatar.png ',
  publicDiceNote: ' 留言 ',
  publicDiceBrief: ' 简介 ',
}, [' ep-2 ', 'ep-1', 'ep-1', '']), {
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
assertEqual(isPublicDiceDirty(baseDraft, { ...baseDraft, selectedEndpointIds: ['ep-2', 'ep-1'] }), false);
assertEqual(isPublicDiceDirty(baseDraft, { ...baseDraft, selectedEndpointIds: ['ep-1'] }), true);

assertDeepEqual(getEndpointStateMeta(1), { text: '已连接', tagType: 'success' });
assertDeepEqual(getEndpointStateMeta(2), { text: '连接中', tagType: 'warning' });
assertDeepEqual(getEndpointStateMeta(3), { text: '失败', tagType: 'error' });
assertDeepEqual(getEndpointStateMeta(0), { text: '断开', tagType: 'error' });

assertEqual(getEndpointProtocolLabel({
  platform: 'QQ',
  protocolType: 'onebot',
  adapter: { builtinMode: 'lagrange' },
}), 'QQ(内置客户端)');
assertEqual(getEndpointProtocolLabel({
  platform: 'QQ',
  protocolType: 'milky',
  adapter: { built_in_mode: 'milky' },
}), 'QQ(内置Milky)');
assertEqual(getEndpointProtocolLabel({
  platform: 'QQ',
  protocolType: 'milky',
}), 'QQ(Milky)');
assertEqual(getEndpointProtocolLabel({
  platform: 'Discord',
  protocolType: 'satori',
}), 'Satori');

assertDeepEqual(getPublicDiceEndpointRows([
  { id: 'ep-1', userId: '1001', platform: 'QQ', protocolType: 'onebot', state: 1, isPublic: true },
  { id: 'ep-2', userId: '1002', platform: 'QQ', protocolType: 'milky', state: 3, isPublic: false },
]), [
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
    protocol: 'QQ(Milky)',
    protocolType: 'milky',
    stateText: '失败',
    stateTagType: 'error',
    isPublic: false,
  },
]);
