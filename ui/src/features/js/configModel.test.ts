import {
  buildConfigErrorKey,
  groupPluginConfigItems,
  isDailyTaskExpressionValid,
  normalizeTemplateValue,
  setConfigError,
  shouldBlockConfigSave,
  type JsConfigErrorMap,
} from './configModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const groups = groupPluginConfigItems([
  { key: 'a', type: 'string', description: '', defaultValue: '', group: '高级' },
  { key: 'b', type: 'string', description: '', defaultValue: '' },
  { key: 'c', type: 'string', description: '', defaultValue: '', group: '高级' },
]);
assertEqual(groups.length, 2);
assertEqual(groups[0]?.name, '高级');
assertDeepEqual(groups[0]?.items.map(item => item.key), ['a', 'c']);
assertEqual(groups[1]?.name, '默认');
assertDeepEqual(normalizeTemplateValue(['a', 1, null, 'b']), ['a', '1', 'b']);
assertDeepEqual(normalizeTemplateValue('single'), ['single']);
assertDeepEqual(normalizeTemplateValue(null), ['']);
assertEqual(isDailyTaskExpressionValid('00:00'), true);
assertEqual(isDailyTaskExpressionValid('23:59'), true);
assertEqual(isDailyTaskExpressionValid('24:00'), false);
assertEqual(buildConfigErrorKey('plugin', 'cron'), 'plugin/cron');

const errors: JsConfigErrorMap = {};
setConfigError(errors, 'plugin', 'cron', '格式错误');
assertEqual(shouldBlockConfigSave(errors), true);
setConfigError(errors, 'plugin', 'cron', '');
assertEqual(shouldBlockConfigSave(errors), false);
