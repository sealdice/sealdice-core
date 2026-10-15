import {
  buildConfigErrorKey,
  groupPluginConfigItems,
  isDailyTaskExpressionValid,
  normalizeTemplateValue,
  setConfigError,
  shouldBlockConfigSave,
  type JsConfigErrorMap,
} from './configModel.js';
import { expect, it } from 'vitest';

it('脚本配置按类型归一化并构造更新数据', async () => {
  const groups = groupPluginConfigItems([
    { key: 'a', type: 'string', description: '', defaultValue: '', group: '高级' },
    { key: 'b', type: 'string', description: '', defaultValue: '' },
    { key: 'c', type: 'string', description: '', defaultValue: '', group: '高级' },
  ]);
  expect(groups.length).toBe(2);
  expect(groups[0]?.name).toBe('高级');
  expect(groups[0]?.items.map(item => item.key)).toEqual(['a', 'c']);
  expect(groups[1]?.name).toBe('默认');
  expect(normalizeTemplateValue(['a', 1, null, 'b'])).toEqual(['a', '1', 'b']);
  expect(normalizeTemplateValue('single')).toEqual(['single']);
  expect(normalizeTemplateValue(null)).toEqual(['']);
  expect(isDailyTaskExpressionValid('00:00')).toBe(true);
  expect(isDailyTaskExpressionValid('23:59')).toBe(true);
  expect(isDailyTaskExpressionValid('24:00')).toBe(false);
  expect(isDailyTaskExpressionValid(' 08:00 ')).toBe(false);
  expect(buildConfigErrorKey('plugin', 'cron')).toBe('plugin/cron');

  const errors: JsConfigErrorMap = {};
  setConfigError(errors, 'plugin', 'cron', '格式错误');
  expect(shouldBlockConfigSave(errors)).toBe(true);
  setConfigError(errors, 'plugin', 'cron', '');
  expect(shouldBlockConfigSave(errors)).toBe(false);
});
