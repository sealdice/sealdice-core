export interface BuildSignInfoStateInput {
  selectedProtocolKey: string;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  hasData: boolean;
  signServerVersion: string;
}

export interface SignInfoState {
  visible: boolean;
  mode: 'hidden' | 'loading' | 'manual-fallback' | 'ready';
  canSelectVersion: boolean;
  canSelectServer: boolean;
  showCustomServerInput: boolean;
  showRetry: boolean;
  message: string;
}

export function buildSignInfoState(input: BuildSignInfoStateInput): SignInfoState {
  if (input.selectedProtocolKey !== 'lagrange') {
    return {
      visible: false,
      mode: 'hidden',
      canSelectVersion: false,
      canSelectServer: false,
      showCustomServerInput: false,
      showRetry: false,
      message: '',
    };
  }

  if (input.isLoading && !input.hasData) {
    return {
      visible: true,
      mode: 'loading',
      canSelectVersion: false,
      canSelectServer: false,
      showCustomServerInput: false,
      showRetry: false,
      message: '正在加载签名服务信息…',
    };
  }

  if (input.isError && !input.hasData) {
    return {
      visible: true,
      mode: 'manual-fallback',
      canSelectVersion: true,
      canSelectServer: false,
      showCustomServerInput: true,
      showRetry: true,
      message: '签名服务信息读取失败，可重试或手动填写签名地址。',
    };
  }

  const manualVersion = input.signServerVersion === '自定义';

  return {
    visible: true,
    mode: 'ready',
    canSelectVersion: true,
    canSelectServer: !manualVersion,
    showCustomServerInput: manualVersion,
    showRetry: false,
    message: input.isFetching ? '正在刷新签名服务信息…' : '',
  };
}
