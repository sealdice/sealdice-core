import type { FormConfigItem, MethodTreeNode, PlatformTreeNode, ProtocolDefinition } from '@/api';
import {
  buildDynamicFormInitialModel,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { validateOfficialQQConfig, type OfficialQQMode } from './officialQQ';

export type ConnectWizardState = {
  step: number;
  platform: PlatformTreeNode | null;
  method: MethodTreeNode | null;
  protocol: ProtocolDefinition | null;
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
    formModel: {},
    officialQQMode: 'manual',
  };
}

export function selectConnectPlatform(state: ConnectWizardState, platform: PlatformTreeNode): void {
  state.platform = platform;
  state.method = null;
  state.protocol = null;
  state.formModel = {};
}

export function selectConnectMethod(state: ConnectWizardState, method: MethodTreeNode): void {
  state.method = method;
  state.protocol = null;
  state.formModel = {};
}

export function selectConnectProtocol(
  state: ConnectWizardState,
  protocol: ProtocolDefinition
): void {
  state.protocol = protocol;
}

export function advanceConnectWizard(
  state: ConnectWizardState,
  selectedSchema: FormConfigItem[]
): void {
  if (state.step === 3 && state.protocol) {
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
  return state.protocol.key === 'officialqq'
    ? validateOfficialQQConfig(state.formModel, state.officialQQMode)
    : { valid: true, message: '' };
}
