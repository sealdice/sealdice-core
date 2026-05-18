import {
  getLegacyAccessToken,
  legacyRequest,
  setLegacyAccessToken,
} from './client';
import {
  checkSecurity,
  getBaseInfo,
  getHello,
  getLogFetchAndClear,
  getPreInfo,
  type DiceBaseInfo,
  type PreInfo,
  type SysLog,
} from '../../features/runtime/legacyApi';

async function assertLegacyClientContract() {
  setLegacyAccessToken('token-value');
  const token: string = getLegacyAccessToken();
  const payload = await legacyRequest<{ ok: boolean }>('get', '/hello');
  const ok: boolean = payload.ok;

  void token;
  void ok;
}

async function assertRuntimeApiContract() {
  const preInfo: PreInfo = await getPreInfo();
  const baseInfo: DiceBaseInfo = await getBaseInfo();
  const logs: SysLog[] = await getLogFetchAndClear();
  const hello: unknown = await getHello();
  const security = await checkSecurity();

  const testMode: boolean = preInfo.testMode;
  const version: string = baseInfo.version;
  const isOk: boolean = security.isOk;

  void logs;
  void hello;
  void testMode;
  void version;
  void isOk;
}

void assertLegacyClientContract;
void assertRuntimeApiContract;
