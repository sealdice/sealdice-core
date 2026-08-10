import { it } from 'vitest';
import { getConnectProtocolModule } from './index';

it('uses specialized modules for known protocols and generic fallback for unknown protocols', () => {
  const officialQQ = getConnectProtocolModule('officialqq');
  const lagrange = getConnectProtocolModule('lagrange');
  const milky = getConnectProtocolModule('milky-aaa');

  if (officialQQ.formKind !== 'officialqq') {
    throw new Error(`officialqq form kind = ${officialQQ.formKind}`);
  }
  if (lagrange.formKind !== 'sign-info') {
    throw new Error(`lagrange form kind = ${lagrange.formKind}`);
  }
  if (milky.key !== 'milky-aaa' || milky.formKind !== 'generic') {
    throw new Error(`unexpected fallback module = ${JSON.stringify(milky)}`);
  }
});
