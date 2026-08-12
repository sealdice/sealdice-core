import type { FormConfigItem, PlatformTreeNode, ProtocolDefinition } from '@/api';
import { it } from 'vitest';
import {
  advanceConnectWizard,
  buildConnectCreatePayload,
  createConnectWizardState,
  selectConnectMethod,
  selectConnectPlatform,
  selectConnectProtocol,
  validateConnectWizardForm,
} from './createWizard';

const platform: PlatformTreeNode = {
  id: 'qq',
  name: 'QQ',
  description: '',
  methods: [],
};

const protocol: ProtocolDefinition = {
  key: 'officialqq',
  name: 'QQ 官方机器人',
  platform: 'QQ',
  schemaKey: 'officialqq',
  deprecated: false,
  available: true,
  capabilities: {
    create: true,
    update: true,
    delete: true,
    enable: true,
    workflow: true,
    qrcode: true,
    signInfo: false,
  },
};

it('clears dependent selections when the wizard parent selection changes', () => {
  const state = createConnectWizardState();
  state.method = { id: 'default', name: '默认', description: '', protocols: [protocol] };
  state.protocol = protocol;
  state.selectedProtocolKey = protocol.key;

  selectConnectPlatform(state, platform);
  if (state.method !== null || state.protocol !== null || state.selectedProtocolKey !== '') {
    throw new Error(`platform selection did not clear children: ${JSON.stringify(state)}`);
  }

  state.protocol = protocol;
  state.selectedProtocolKey = protocol.key;
  selectConnectMethod(state, { id: 'default', name: '默认', description: '', protocols: [protocol] });
  if (state.protocol !== null || state.selectedProtocolKey !== '') {
    throw new Error(`method selection did not clear protocol: ${JSON.stringify(state)}`);
  }
});

it('initializes protocol form and advances to the form step', () => {
  const state = createConnectWizardState();
  state.step = 3;
  const schema: FormConfigItem[] = [{
    check_type: 0,
    default: '',
    err_msg: '',
    field_name: 'appID',
    hint: '',
    id: 1,
    input_type: 0,
    is_required: 0,
    name: '机器人 ID',
    placeholder: '',
    size_range: { max: 0, min: 0 },
    sub_option: null,
  }];
  selectConnectProtocol(state, protocol, schema);
  advanceConnectWizard(state, schema);
  if (state.step !== 4 || state.selectedProtocolKey !== 'officialqq' || state.formModel.appID !== '') {
    throw new Error(`unexpected wizard state = ${JSON.stringify(state)}`);
  }
});

it('uses the protocol module to validate and prepare a create payload', () => {
  const state = createConnectWizardState();
  state.protocol = protocol;
  state.selectedProtocolKey = protocol.key;
  state.formModel = { appID: '10001', appSecret: 'secret' };
  state.officialQQMode = 'qrcode';

  const validation = validateConnectWizardForm(state, []);
  if (!validation.valid) throw new Error(validation.message);
  const payload = buildConnectCreatePayload(state);
  if (JSON.stringify(payload) !== JSON.stringify({
    platform: 'officialqq',
    config: { appID: '', appSecret: '', useWebhook: false },
  })) {
    throw new Error(`unexpected create payload = ${JSON.stringify(payload)}`);
  }
});
