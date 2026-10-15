import type { FormConfigItem, PlatformTreeNode, ProtocolDefinition } from '@/api';
import { it } from 'vitest';
import {
  advanceConnectWizard,
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

  selectConnectPlatform(state, platform);
  if (state.method !== null || state.protocol !== null) {
    throw new Error(`platform selection did not clear children: ${JSON.stringify(state)}`);
  }

  state.protocol = protocol;
  selectConnectMethod(state, {
    id: 'default',
    name: '默认',
    description: '',
    protocols: [protocol],
  });
  if (state.protocol !== null) {
    throw new Error(`method selection did not clear protocol: ${JSON.stringify(state)}`);
  }
});

it('initializes protocol form and advances to the form step', () => {
  const state = createConnectWizardState();
  state.step = 3;
  const schema: FormConfigItem[] = [
    {
      check_type: 0,
      default: '',
      field_name: 'appID',
      input_type: 0,
      is_required: 0,
      name: '机器人 ID',
      placeholder: '',
      sub_option: null,
    },
  ];
  selectConnectProtocol(state, protocol);
  advanceConnectWizard(state, schema);
  if (state.step !== 4 || state.protocol?.key !== 'officialqq' || state.formModel.appID !== '') {
    throw new Error(`unexpected wizard state = ${JSON.stringify(state)}`);
  }
});

it('validates official QQ QR mode without requiring manual credentials', () => {
  const state = createConnectWizardState();
  state.protocol = protocol;
  state.formModel = { appID: '10001', appSecret: 'secret' };
  state.officialQQMode = 'qrcode';

  const validation = validateConnectWizardForm(state, []);
  if (!validation.valid) throw new Error(validation.message);
});
