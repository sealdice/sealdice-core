import {
  buildOfficialQQCreateConfig,
  buildOfficialQQDuplicateMessage,
  validateOfficialQQConfig,
} from '../officialQQ';
import type { ConnectProtocolModule } from './generic';

export const officialQQProtocolModule: ConnectProtocolModule = {
  key: 'officialqq',
  formKind: 'officialqq',
  prepareCreateConfig: (config, context) =>
    buildOfficialQQCreateConfig(config, context.officialQQMode ?? 'manual'),
  validateCreate: (config, context) =>
    validateOfficialQQConfig(config, context.officialQQMode ?? 'manual'),
  beforeCreate: async (config, context) => {
    const mode = context.officialQQMode ?? 'manual';
    const validation = validateOfficialQQConfig(config, mode);
    if (!validation.valid) throw new Error(validation.message);
    if (mode === 'qrcode') return;
    if (!context.testOfficialQQ) throw new Error('官方 QQ 连接测试不可用');
    const result = await context.testOfficialQQ(config);
    if (result.exists) throw new Error(buildOfficialQQDuplicateMessage(result.userId));
  },
};
