import {
  buildOfficialQQCreateConfig,
  buildOfficialQQDuplicateMessage,
  validateOfficialQQManualConfig,
} from '../officialQQ';
import type { ConnectProtocolModule } from './generic';

export const officialQQProtocolModule: ConnectProtocolModule = {
  key: 'officialqq',
  formKind: 'officialqq',
  prepareCreateConfig: (config, context) =>
    buildOfficialQQCreateConfig(config, context.officialQQMode ?? 'manual'),
  validateCreate: (config, context) => {
    if (context.officialQQMode !== 'manual') return { valid: true, message: '' };
    return validateOfficialQQManualConfig(config);
  },
  beforeCreate: async (config, context) => {
    if (context.officialQQMode !== 'manual') return;
    const validation = validateOfficialQQManualConfig(config);
    if (!validation.valid) throw new Error(validation.message);
    if (!context.testOfficialQQ) throw new Error('官方 QQ 连接测试不可用');
    const result = await context.testOfficialQQ(config);
    if (result.exists) throw new Error(buildOfficialQQDuplicateMessage(result.userId));
  },
};
