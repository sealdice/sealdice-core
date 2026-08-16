import { lagrangeProtocolModule } from './lagrange';
import { genericProtocolModule, type ConnectProtocolModule } from './generic';
import { officialQQProtocolModule } from './officialqq';

const specializedProtocolModules: Record<string, ConnectProtocolModule> = {
  lagrange: lagrangeProtocolModule,
  officialqq: officialQQProtocolModule,
};

export function getConnectProtocolModule(key: string): ConnectProtocolModule {
  return specializedProtocolModules[key] ?? genericProtocolModule(key);
}
