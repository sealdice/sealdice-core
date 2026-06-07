export type EndpointDisplayAdapter = {
  builtinMode?: string;
  built_in_mode?: string;
  reverseAddr?: string;
};

export type EndpointDisplaySource = {
  platform: string;
  protocolType: string;
  adapter?: EndpointDisplayAdapter | null;
};

export function getEndpointStateMeta(state: number) {
  switch (state) {
    case 1:
      return { text: '已连接', tagType: 'success' as const };
    case 2:
      return { text: '连接中', tagType: 'warning' as const };
    case 3:
      return { text: '失败', tagType: 'error' as const };
    default:
      return { text: '断开', tagType: 'error' as const };
  }
}

export function getEndpointProtocolLabel(endpoint: EndpointDisplaySource) {
  const adapter = endpoint.adapter ?? {};
  if (endpoint.protocolType === 'onebot' && adapter.builtinMode === 'lagrange') return 'QQ(内置客户端)';
  if (endpoint.protocolType === 'milky' && adapter.built_in_mode) return 'QQ(内置Milky)';
  if (endpoint.protocolType === 'milky') return 'QQ(Milky)';
  if (endpoint.protocolType === 'pureonebot' && adapter.reverseAddr) return 'QQ(onebot11反向WS)';
  if (endpoint.protocolType === 'pureonebot') return 'QQ(onebot11正向WS)';
  if (endpoint.protocolType === 'satori') return 'Satori';
  return endpoint.platform;
}
