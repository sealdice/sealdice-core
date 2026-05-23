import type { ConfigItem } from '@/api';

export type ConfigItemGroup = {
  name: string;
  items: ConfigItem[];
};

export type JsConfigErrorMap = Record<string, string>;

const DEFAULT_GROUP = '默认';
const dailyTaskPattern = /^([0-1]?[0-9]|2[0-3]):([0-5][0-9])$/;

export function groupPluginConfigItems(items: ConfigItem[]): ConfigItemGroup[] {
  const groups = new Map<string, ConfigItem[]>();
  for (const item of items) {
    const name = typeof item.group === 'string' && item.group.trim() ? item.group.trim() : DEFAULT_GROUP;
    groups.set(name, [...(groups.get(name) ?? []), item]);
  }
  return Array.from(groups.entries()).map(([name, groupItems]) => ({
    name,
    items: groupItems,
  }));
}

export function normalizeTemplateValue(value: unknown): string[] {
  if (Array.isArray(value)) {
    const normalized = value
      .filter(item => item !== null && item !== undefined)
      .map(item => String(item));
    return normalized.length ? normalized : [''];
  }
  if (value === null || value === undefined) return [''];
  return [String(value)];
}

export function isDailyTaskExpressionValid(expr: string): boolean {
  return dailyTaskPattern.test(expr.trim());
}

export function buildConfigErrorKey(pluginName: string, key: string): string {
  return `${pluginName}/${key}`;
}

export function setConfigError(
  errors: JsConfigErrorMap,
  pluginName: string,
  key: string,
  message: string,
): void {
  const errorKey = buildConfigErrorKey(pluginName, key);
  if (!message) {
    delete errors[errorKey];
    return;
  }
  errors[errorKey] = message;
}

export function shouldBlockConfigSave(errors: JsConfigErrorMap): boolean {
  return Object.keys(errors).length > 0;
}
