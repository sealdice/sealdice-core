import dayjs from 'dayjs';
import type { EndPointInfo, WorkflowResp } from '@/api';

export type EndpointDisplayAdapter = {
  connectUrl?: string;
  builtinMode?: string;
  built_in_mode?: string;
  reverseAddr?: string;
  useWebhook?: boolean;
  ws_gateway?: string;
  signServerVer?: string;
  signServerName?: string;
  webhookPath?: string;
  webhookPort?: number;
  host?: string;
  port?: number;
};

export type EndpointDisplaySource = {
  platform: string;
  protocolType: string;
  adapter?: EndpointDisplayAdapter | null;
};

export type EndpointDetailRow = [label: string, value: string];
export type EndpointMetricRow = [label: string, value: string];

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
  if (endpoint.protocolType === 'onebot' && adapter.builtinMode === 'lagrange')
    return 'QQ（内置客户端）';
  if (endpoint.protocolType === 'milky' && adapter.built_in_mode) return 'QQ（内置 Milky）';
  if (endpoint.protocolType === 'milky') return 'QQ（Milky）';
  if (endpoint.protocolType === 'pureonebot' && adapter.reverseAddr)
    return 'QQ（onebot11 反向 WS）';
  if (endpoint.protocolType === 'pureonebot') return 'QQ（onebot11 正向 WS）';
  if (endpoint.protocolType === 'official')
    return adapter.useWebhook ? 'QQ（官方机器人 Webhook）' : 'QQ（官方机器人）';
  if (endpoint.protocolType === 'satori') return 'Satori';
  return endpoint.platform;
}

export function getEndpointTargetLabel(endpoint: EndPointInfo) {
  const account = endpoint.nickname || endpoint.userId || endpoint.id;
  const protocol = getEndpointProtocolLabel({
    platform: endpoint.platform,
    protocolType: endpoint.protocolType,
    adapter: adapterOf(endpoint),
  });
  return `${account}（${protocol}，ID：${endpoint.id}）`;
}

export function adapterOf(endpoint: EndPointInfo): EndpointDisplayAdapter {
  if (endpoint.adapter && typeof endpoint.adapter === 'object') {
    return endpoint.adapter as EndpointDisplayAdapter;
  }
  return {};
}

export function getWorkflowTag(workflow: WorkflowResp | null) {
  switch (workflow?.state) {
    case 'qrcode':
      return { type: 'warning' as const, text: '等待扫码' };
    case 'pending':
      return { type: 'info' as const, text: '登录中' };
    case 'failed':
      return { type: 'error' as const, text: '登录失败' };
    default:
      return null;
  }
}

export function getWorkflowText(workflow: WorkflowResp | null): string {
  switch (workflow?.state) {
    case 'qrcode':
      return '等待扫码';
    case 'pending':
      return '登录中';
    case 'success':
      return '登录成功';
    case 'failed':
      return workflow.failedReason ? `登录失败：${workflow.failedReason}` : '登录失败';
    default:
      return '';
  }
}

export function getEndpointMetricRows(endpoint: EndPointInfo): EndpointMetricRow[] {
  return [
    ['群组数量', String(endpoint.groupNum)],
    ['累计响应指令', String(endpoint.cmdExecutedNum)],
    [
      '上次执行指令',
      endpoint.cmdExecutedLastTime > 0
        ? dayjs.unix(endpoint.cmdExecutedLastTime).format('YYYY-MM-DD HH:mm:ss')
        : '尚无记录',
    ],
  ];
}

export function getEndpointDetailRows(
  endpoint: EndPointInfo,
  workflow: WorkflowResp | null
): EndpointDetailRow[] {
  const adapter = adapterOf(endpoint);
  return [
    ['账号', endpoint.userId],
    ['登录流程', getWorkflowText(workflow)],
    ['连接地址', adapter.connectUrl || adapter.ws_gateway || ''],
    ['服务地址', adapter.reverseAddr ? `${adapter.reverseAddr}/ws` : ''],
    ['签名版本', adapter.signServerVer || ''],
    ['签名服务', adapter.signServerName || ''],
    ['协议端', adapter.built_in_mode || adapter.builtinMode || ''],
    [
      '接入方式',
      endpoint.protocolType === 'official' ? (adapter.useWebhook ? 'Webhook' : 'WebSocket') : '',
    ],
    ['Webhook 路径', adapter.useWebhook ? adapter.webhookPath || '' : ''],
    ['Webhook 端口', adapter.useWebhook && adapter.webhookPort ? String(adapter.webhookPort) : ''],
    ['主机', adapter.host ? `${adapter.host}${adapter.port ? `:${adapter.port}` : ''}` : ''],
  ].filter(([, value]) => value) as EndpointDetailRow[];
}
