import { buildSignInfoState } from './signInfoState.ts';
import { expect, it } from 'vitest';

it('签名服务返回空列表时允许手动输入', () => {
  const state = buildSignInfoState({
    enabled: true,
    isLoading: false,
    isFetching: false,
    isError: false,
    hasData: false,
    signServerVersion: '',
  });
  expect(state.mode).toBe('manual-fallback');
  expect(state.showCustomServerInput).toBe(true);
  expect(state.showRetry).toBe(true);
});

it('签名服务选择保留合法选项', async () => {
  expect(
    buildSignInfoState({
      enabled: false,
      isLoading: false,
      isFetching: false,
      isError: false,
      hasData: false,
      signServerVersion: '',
    })
  ).toEqual({
    visible: false,
    mode: 'hidden',
    canSelectVersion: false,
    canSelectServer: false,
    showCustomServerInput: false,
    showRetry: false,
    message: '',
  });

  expect(
    buildSignInfoState({
      enabled: true,
      isLoading: true,
      isFetching: true,
      isError: false,
      hasData: false,
      signServerVersion: '',
    })
  ).toEqual({
    visible: true,
    mode: 'loading',
    canSelectVersion: false,
    canSelectServer: false,
    showCustomServerInput: false,
    showRetry: false,
    message: '正在加载签名服务信息…',
  });

  expect(
    buildSignInfoState({
      enabled: true,
      isLoading: false,
      isFetching: false,
      isError: true,
      hasData: false,
      signServerVersion: '',
    })
  ).toEqual({
    visible: true,
    mode: 'manual-fallback',
    canSelectVersion: true,
    canSelectServer: false,
    showCustomServerInput: true,
    showRetry: true,
    message: '签名服务信息读取失败，可重试或手动填写签名地址。',
  });

  const readyAuto = buildSignInfoState({
    enabled: true,
    isLoading: false,
    isFetching: false,
    isError: false,
    hasData: true,
    signServerVersion: '30366',
  });
  expect(readyAuto.visible).toBe(true);
  expect(readyAuto.mode).toBe('ready');
  expect(readyAuto.canSelectVersion).toBe(true);
  expect(readyAuto.canSelectServer).toBe(true);
  expect(readyAuto.showCustomServerInput).toBe(false);
  expect(readyAuto.showRetry).toBe(false);

  const readyManual = buildSignInfoState({
    enabled: true,
    isLoading: false,
    isFetching: false,
    isError: false,
    hasData: true,
    signServerVersion: '自定义',
  });
  expect(readyManual.visible).toBe(true);
  expect(readyManual.mode).toBe('ready');
  expect(readyManual.canSelectVersion).toBe(true);
  expect(readyManual.canSelectServer).toBe(false);
  expect(readyManual.showCustomServerInput).toBe(true);
});
