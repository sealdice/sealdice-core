import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
import type { OfficialQQMode } from '../officialQQ';

export type ConnectProtocolFormKind = 'generic' | 'officialqq' | 'sign-info';

export type ConnectProtocolCreateContext = {
  officialQQMode?: OfficialQQMode;
  testOfficialQQ?: (
    config: Record<string, unknown>
  ) => Promise<{ exists: boolean; userId: string }>;
};

export type ConnectProtocolModule = {
  key: string;
  formKind: ConnectProtocolFormKind;
  prepareCreateConfig?: (
    config: DynamicFormModel,
    context: ConnectProtocolCreateContext
  ) => DynamicFormModel;
  validateCreate?: (
    config: DynamicFormModel,
    context: ConnectProtocolCreateContext
  ) => { valid: boolean; message: string };
  beforeCreate?: (config: DynamicFormModel, context: ConnectProtocolCreateContext) => Promise<void>;
};

export function genericProtocolModule(key: string): ConnectProtocolModule {
  return { key, formKind: 'generic' };
}
