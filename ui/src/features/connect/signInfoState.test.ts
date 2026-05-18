import { buildSignInfoState } from './signInfoState.ts';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(
  buildSignInfoState({
    selectedProtocolKey: 'qq',
    isLoading: false,
    isFetching: false,
    isError: false,
    hasData: false,
    signServerVersion: '',
  }),
  {
    visible: false,
    mode: 'hidden',
    canSelectVersion: false,
    canSelectServer: false,
    showCustomServerInput: false,
    showRetry: false,
    message: '',
  },
);

assertDeepEqual(
  buildSignInfoState({
    selectedProtocolKey: 'lagrange',
    isLoading: true,
    isFetching: true,
    isError: false,
    hasData: false,
    signServerVersion: '',
  }),
  {
    visible: true,
    mode: 'loading',
    canSelectVersion: false,
    canSelectServer: false,
    showCustomServerInput: false,
    showRetry: false,
    message: '正在加载签名服务信息…',
  },
);

assertDeepEqual(
  buildSignInfoState({
    selectedProtocolKey: 'lagrange',
    isLoading: false,
    isFetching: false,
    isError: true,
    hasData: false,
    signServerVersion: '',
  }),
  {
    visible: true,
    mode: 'manual-fallback',
    canSelectVersion: true,
    canSelectServer: false,
    showCustomServerInput: true,
    showRetry: true,
    message: '签名服务信息读取失败，可重试或手动填写签名地址。',
  },
);

const readyAuto = buildSignInfoState({
  selectedProtocolKey: 'lagrange',
  isLoading: false,
  isFetching: false,
  isError: false,
  hasData: true,
  signServerVersion: '30366',
});
assertEqual(readyAuto.visible, true);
assertEqual(readyAuto.mode, 'ready');
assertEqual(readyAuto.canSelectVersion, true);
assertEqual(readyAuto.canSelectServer, true);
assertEqual(readyAuto.showCustomServerInput, false);
assertEqual(readyAuto.showRetry, false);

const readyManual = buildSignInfoState({
  selectedProtocolKey: 'lagrange',
  isLoading: false,
  isFetching: false,
  isError: false,
  hasData: true,
  signServerVersion: '自定义',
});
assertEqual(readyManual.visible, true);
assertEqual(readyManual.mode, 'ready');
assertEqual(readyManual.canSelectVersion, true);
assertEqual(readyManual.canSelectServer, false);
assertEqual(readyManual.showCustomServerInput, true);
