import type { EndPointInfo, WorkflowResp } from '@/api';
import { it } from 'vitest';
import {
  getEndpointDetailRows,
  getWorkflowTag,
  getWorkflowText,
  getEndpointTargetLabel,
} from './endpointDisplay';

const endpoint: EndPointInfo = {
  adapter: {
    connectUrl: 'ws://127.0.0.1:3000',
    signServerVer: '30366',
    signServerName: 'sealdice',
  },
  cmdExecutedLastTime: 0,
  cmdExecutedNum: 3,
  enable: true,
  groupNum: 2,
  id: 'ep-1',
  isPublic: false,
  nickname: '',
  onlineTotalTime: 0,
  platform: 'QQ',
  protocolType: 'onebot',
  relWorkDir: '',
  state: 1,
  userId: 'QQ:10001',
};

it('maps workflow states to tags and detailed text', () => {
  const workflow: WorkflowResp = {
    state: 'failed',
    failedReason: 'invalid token',
    hasQRCode: false,
    loginState: 0,
  };
  const tag = getWorkflowTag(workflow);
  if (tag?.text !== '登录失败' || tag.type !== 'error') throw new Error(`unexpected tag = ${JSON.stringify(tag)}`);
  if (getWorkflowText(workflow) !== '登录失败：invalid token') {
    throw new Error(`unexpected workflow text = ${getWorkflowText(workflow)}`);
  }
});

it('builds endpoint detail rows without platform logic in the page', () => {
  const rows = getEndpointDetailRows(endpoint, { state: 'qrcode', hasQRCode: true, loginState: 0 });
  const rowMap = new Map(rows);
  if (rowMap.get('账号') !== 'QQ:10001') throw new Error(`unexpected account = ${rowMap.get('账号')}`);
  if (rowMap.get('连接地址') !== 'ws://127.0.0.1:3000') {
    throw new Error(`unexpected connect url = ${rowMap.get('连接地址')}`);
  }
  if (rowMap.get('签名服务') !== 'sealdice') throw new Error(`unexpected sign server = ${rowMap.get('签名服务')}`);
});

if (getEndpointTargetLabel(endpoint) !== 'QQ:10001（QQ，ID: ep-1）') {
  throw new Error(`unexpected endpoint target = ${getEndpointTargetLabel(endpoint)}`);
}
