import type { EndPointInfo, WorkflowResp } from '@/api';
import {
  applyConnectionList,
  applyConnectionSnapshot,
  applyConnectionQRCode,
  applyConnectionUpdate,
  applyConnectionWorkflow,
} from './realtimeState';
import { expect, it } from 'vitest';

it('连接变更清理关联工作流和二维码', async () => {
  const connA: EndPointInfo = {
    adapter: {},
    cmdExecutedLastTime: 0,
    cmdExecutedNum: 0,
    enable: true,
    groupNum: 0,
    id: 'a',
    isPublic: false,
    nickname: 'A',
    onlineTotalTime: 0,
    platform: 'QQ',
    protocolType: 'milky',
    relWorkDir: '',
    state: 2,
    userId: 'QQ:10001',
  };

  const connB: EndPointInfo = {
    ...connA,
    id: 'b',
    nickname: 'B',
    userId: 'QQ:10002',
  };

  const workflow: WorkflowResp = {
    state: 'qrcode',
    hasQRCode: true,
    loginState: 2,
  };

  expect(
    applyConnectionList(
      [connA],
      { a: workflow, stale: { state: 'failed', hasQRCode: false, loginState: 11 } },
      { a: 'data:image/png;base64,abc', stale: 'old' },
      [connB]
    )
  ).toEqual({
    connections: [connB],
    workflows: {},
    qrCodes: {},
  });

  expect(
    applyConnectionSnapshot(
      [connA],
      { a: workflow, stale: { state: 'failed', hasQRCode: false, loginState: 11 } },
      { a: 'data:image/png;base64,abc', stale: 'old' },
      [connB]
    )
  ).toEqual({
    connections: [connB],
    workflows: {},
    qrCodes: {},
    ready: true,
  });

  expect(applyConnectionUpdate([connA], connB)).toEqual([connA, connB]);
  expect(applyConnectionUpdate([connA], { ...connA, nickname: 'A2' })[0]?.nickname).toEqual('A2');

  expect(applyConnectionWorkflow({}, 'a', workflow)).toEqual({ a: workflow });
  expect(applyConnectionQRCode({}, 'a', 'data:image/png;base64,abc')).toEqual({
    a: 'data:image/png;base64,abc',
  });
  expect(applyConnectionQRCode({ a: 'x' }, 'a', '')).toEqual({});
});
