import type { FormConfigItem, MethodTreeNode, PlatformTreeNode, ProtocolDefinition } from '@/api';
import {
  buildDynamicFormInitialModel,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { getConnectProtocolModule } from './protocols';
import type { OfficialQQMode } from './officialQQ';

export type ConnectWizardState = {
  step: number;
  platform: PlatformTreeNode | null;
  method: MethodTreeNode | null;
  protocol: ProtocolDefinition | null;
  selectedProtocolKey: string;
  formModel: DynamicFormModel;
  officialQQMode: OfficialQQMode;
};

export type ConnectWizardValidation = {
  valid: boolean;
  message: string;
};

export function createConnectWizardState(): ConnectWizardState {
  return {
    step: 1,
    platform: null,
    method: null,
    protocol: null,
    selectedProtocolKey: '',
    formModel: {},
    officialQQMode: 'manual',
  };
}

export function selectConnectPlatform(state: ConnectWizardState, platform: PlatformTreeNode): void {
  state.platform = platform;
  state.method = null;
  state.protocol = null;
  state.selectedProtocolKey = '';
  state.formModel = {};
}

export function selectConnectMethod(state: ConnectWizardState, method: MethodTreeNode): void {
  state.method = method;
  state.protocol = null;
  state.selectedProtocolKey = '';
  state.formModel = {};
}

export function selectConnectProtocol(
  state: ConnectWizardState,
  protocol: ProtocolDefinition,
  _schema: FormConfigItem[]
): void {
  state.protocol = protocol;
  state.selectedProtocolKey = protocol.key;
}

export function advanceConnectWizard(
  state: ConnectWizardState,
  selectedSchema: FormConfigItem[]
): void {
  if (state.step === 3 && state.protocol) {
    state.selectedProtocolKey = state.protocol.key;
    state.formModel = buildDynamicFormInitialModel(selectedSchema);
    state.officialQQMode = 'manual';
  }
  if (state.step < 4) state.step += 1;
}

export function resetConnectWizard(state: ConnectWizardState): void {
  Object.assign(state, createConnectWizardState());
}

export function validateConnectWizardForm(
  state: ConnectWizardState,
  selectedSchema: FormConfigItem[]
): ConnectWizardValidation {
  if (!state.protocol?.available || state.protocol.deprecated) {
    return { valid: false, message: state.protocol?.disabledReason || '当前协议不可用' };
  }
  const formValidation = validateDynamicFormModel(selectedSchema, state.formModel);
  if (!formValidation.valid) {
    return { valid: false, message: `请填写必填项：${formValidation.missingFields.join('、')}` };
  }
  const protocolValidation = getConnectProtocolModule(state.selectedProtocolKey).validateCreate?.(
    state.formModel,
    { officialQQMode: state.officialQQMode }
  );
  return protocolValidation ?? { valid: true, message: '' };
}

export function buildConnectCreatePayload(state: ConnectWizardState): {
  platform: string;
  config: DynamicFormModel;
} {
  const module = getConnectProtocolModule(state.selectedProtocolKey);
  const config = module.prepareCreateConfig?.(
    { ...state.formModel },
    { officialQQMode: state.officialQQMode }
  ) ?? { ...state.formModel };
  return {
    platform: state.selectedProtocolKey,
    config,
  };
}
