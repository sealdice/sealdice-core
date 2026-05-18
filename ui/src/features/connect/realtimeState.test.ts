import type { EndPointInfo, WorkflowResp } from '@/api';
import {
  applyConnectionList,
  applyConnectionQRCode,
  applyConnectionUpdate,
  applyConnectionWorkflow,
} from './realtimeState';

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const connA: EndPointInfo = {
  adapter: null,
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

assertDeepEqual(
  applyConnectionList(
    [connA],
    { a: workflow, stale: { state: 'failed', hasQRCode: false, loginState: 11 } },
    { a: 'data:image/png;base64,abc', stale: 'old' },
    [connB],
  ),
  {
    connections: [connB],
    workflows: {},
    qrCodes: {},
  },
);

assertDeepEqual(applyConnectionUpdate([connA], connB), [connA, connB]);
assertDeepEqual(applyConnectionUpdate([connA], { ...connA, nickname: 'A2' })[0]?.nickname, 'A2');

assertDeepEqual(applyConnectionWorkflow({}, 'a', workflow), { a: workflow });
assertDeepEqual(applyConnectionQRCode({}, 'a', 'data:image/png;base64,abc'), { a: 'data:image/png;base64,abc' });
assertDeepEqual(applyConnectionQRCode({ a: 'x' }, 'a', ''), {});
