import { toRaw } from 'vue';
import { cloneDeep } from 'es-toolkit';
import { isArray, isEqual, isObject, transform } from 'es-toolkit/compat';
import type {
  BaseSettingExtDefaultSettingItem,
  BaseSettingFieldSchema,
  BaseSettingGroupSchema,
  BaseSettingSchemaResp,
  BaseSettingTabSchema,
  BaseSettingValueResp,
} from '@/api';

export type BaseSettingValueModel = Omit<
  BaseSettingValueResp,
  'commandPrefix' | 'diceMasters' | 'noticeIds' | 'extDefaultSettings'
> & {
  commandPrefix: string[];
  diceMasters: string[];
  noticeIds: string[];
  extDefaultSettings: BaseSettingExtDefaultSettingItem[];
};

export type BaseSettingFieldModel = Omit<
  BaseSettingFieldSchema,
  'keywords' | 'options' | 'keys'
> & {
  keywords: string[];
  options: NonNullable<BaseSettingFieldSchema['options']>;
  keys: string[];
};

export type BaseSettingGroupModel = Omit<BaseSettingGroupSchema, 'fields' | 'notes'> & {
  fields: BaseSettingFieldModel[];
  notes: NonNullable<BaseSettingGroupSchema['notes']>;
};

export type BaseSettingTabModel = Omit<BaseSettingTabSchema, 'groups'> & {
  groups: BaseSettingGroupModel[];
};

export type BaseSettingSchemaModel = {
  tabs: BaseSettingTabModel[];
};

export type BaseSettingSearchEntry = {
  fieldId: string;
  fieldKey: string;
  label: string;
  hint: string;
  tabId: string;
  tabTitle: string;
  groupId: string;
  groupTitle: string;
  searchText: string;
};

export type BaseSettingFieldLayout = 'inline' | 'auto' | 'stacked';
export type ExtDefaultSettingsFilterMode = 'all' | 'modified';
export type ExtDefaultSettingsSortKey =
  | 'source'
  | 'modified'
  | 'name'
  | 'auto-active'
  | 'disabled-count';

export type ExtDefaultSettingsViewItem = {
  item: BaseSettingExtDefaultSettingItem;
  originalIndex: number;
  commandCount: number;
  disabledCount: number;
  searchText: string;
  dirty: boolean;
  autoActiveDirty: boolean;
  disabledCommandDirty: boolean;
  changedCommands: string[];
};

export function normalizeBaseSettingValue(value: BaseSettingValueResp): BaseSettingValueModel {
  return {
    ...value,
    commandPrefix: [...(value.commandPrefix ?? [])],
    diceMasters: [...(value.diceMasters ?? [])],
    noticeIds: [...(value.noticeIds ?? [])],
    extDefaultSettings: (value.extDefaultSettings ?? []).map(item => ({
      ...item,
      disabledCommand: { ...item.disabledCommand },
    })),
  };
}

// 草稿存在 ref 里，读出来的是响应式代理，structuredClone 会抛 DataCloneError。
// toRaw 只剥一层，updateField 的 spread 会把嵌套层留成代理，因此这里必须用深克隆。
export function cloneBaseSettingValue(value: BaseSettingValueModel): BaseSettingValueModel {
  return cloneDeep(toRaw(value));
}

export function normalizeBaseSettingSchema(schema: BaseSettingSchemaResp): BaseSettingSchemaModel {
  return {
    tabs: (schema.tabs ?? []).map<BaseSettingTabModel>(tab => ({
      ...tab,
      groups: (tab.groups ?? []).map<BaseSettingGroupModel>(group => ({
        ...group,
        notes: group.notes ?? [],
        fields: (group.fields ?? []).map<BaseSettingFieldModel>(field => ({
          ...field,
          keywords: field.keywords ?? [],
          options: field.options ?? [],
          keys: field.keys ?? [],
        })),
      })),
    })),
  };
}

export function buildBaseSettingSearchIndex(
  schema: BaseSettingSchemaModel
): BaseSettingSearchEntry[] {
  const entries: BaseSettingSearchEntry[] = [];
  for (const tab of schema.tabs) {
    for (const group of tab.groups) {
      for (const field of group.fields) {
        const tokens = [tab.title, group.title, field.label, field.hint, ...(field.keywords ?? [])]
          .filter(Boolean)
          .join(' ')
          .toLowerCase();
        entries.push({
          fieldId: field.id,
          fieldKey: field.key || field.keys[0] || field.id,
          label: field.label,
          hint: field.hint || '',
          tabId: tab.id,
          tabTitle: tab.title,
          groupId: group.id,
          groupTitle: group.title,
          searchText: tokens,
        });
      }
    }
  }
  return entries;
}

export function searchBaseSettingFields(index: BaseSettingSearchEntry[], keyword: string) {
  const value = keyword.trim().toLowerCase();
  if (!value) return [] as BaseSettingSearchEntry[];
  return index.filter(item => item.searchText.includes(value)).slice(0, 12);
}

export function isBaseSettingGroupWide(groupId: string) {
  return ['ext-default-settings', 'upgrade', 'rate-limit-main'].includes(groupId);
}

export function getBaseSettingFieldLayout(
  field: Pick<BaseSettingFieldModel, 'kind'>
): BaseSettingFieldLayout {
  if (
    ['ext-default-settings', 'string-list', 'notice-targets', 'upload', 'master-list'].includes(
      field.kind
    )
  )
    return 'stacked';
  if (['boolean', 'action', 'unlock-code'].includes(field.kind)) return 'inline';
  return 'auto';
}

export function isBaseSettingFieldBottomMounted(field: Pick<BaseSettingFieldModel, 'kind'>) {
  return getBaseSettingFieldLayout(field) === 'stacked';
}

export function getBaseSettingFieldFeedback(field: Pick<BaseSettingFieldModel, 'key' | 'hint'>) {
  if (field.key === 'QQEnablePoke') {
    return '启用前请确认你使用的 QQ 连接方式支持该功能，若不支持请关闭该功能来避免日志中出现相关报错。';
  }
  return field.hint || '';
}

export function buildBaseSettingStringListOptions(values: string[]) {
  const seen = new Set<string>();
  return values.reduce<Array<{ label: string; value: string }>>((options, value) => {
    const normalized = value.trim();
    if (!normalized || seen.has(normalized)) return options;
    seen.add(normalized);
    options.push({ label: normalized, value: normalized });
    return options;
  }, []);
}

// 字符串列表在提交前统一走这里：去空白、去空项、去重复项。
// 既防止编辑过程中产生重复条目，也兜底历史数据里可能存在的重复项。
export function normalizeStringListValues(values: string[]) {
  return buildBaseSettingStringListOptions(values).map(option => option.value);
}

// Master 列表复用通用归一化逻辑，保留独立命名便于搜索与测试。
export function normalizeMasterListValues(values: string[]) {
  return normalizeStringListValues(values);
}

// ---- 文本字段格式校验（与后端 utils.ParseRate / robfig-cron 标准解析保持一致） ----

// Go time.ParseDuration 的合法单元组合：数字(可带小数)+单位，可连续拼接（如 1h30m）。
const goDurationPartRe = '\\d+(?:\\.\\d+)?(?:ns|us|µs|ms|s|m|h)';
const goDurationRe = new RegExp('^(?:0|[+-]?(?:' + goDurationPartRe + ')+)$');

export function isEveryDuration(value: string) {
  const trimmed = value.trim();
  const match = /^@every\s+(.+)$/.exec(trimmed);
  return Boolean(match && goDurationRe.test(match[1]!));
}

// 速率：正整数，或 @every 时长。整数 0 与后端 ParseRate(0) 的拒绝语义一致，视为无效。
export function isRateFormatValid(value: string) {
  const trimmed = value.trim();
  if (/^[1-9]\d*$/.test(trimmed)) return true;
  return isEveryDuration(trimmed);
}

const cronDescriptorSet = new Set([
  '@yearly',
  '@annually',
  '@monthly',
  '@weekly',
  '@daily',
  '@midnight',
  '@hourly',
]);

const cronMonthNames: Record<string, number> = {
  jan: 1,
  feb: 2,
  mar: 3,
  apr: 4,
  may: 5,
  jun: 6,
  jul: 7,
  aug: 8,
  sep: 9,
  oct: 10,
  nov: 11,
  dec: 12,
};

const cronDayNames: Record<string, number> = {
  sun: 0,
  mon: 1,
  tue: 2,
  wed: 3,
  thu: 4,
  fri: 5,
  sat: 6,
};

// 标准 cron 五个字段的范围：分 时 日 月 周（robfig-cron 为 0-6）。
const cronFieldBounds: Array<[number, number]> = [
  [0, 59],
  [0, 23],
  [1, 31],
  [1, 12],
  [0, 6],
];

function cronTokenValue(token: string, fieldIndex: number): number | null {
  const trimmed = token.trim();
  if (!trimmed) return null;
  if (/^\d+$/.test(trimmed)) return Number(trimmed);
  if (fieldIndex === 3) return cronMonthNames[trimmed.toLowerCase()] ?? null;
  if (fieldIndex === 4) return cronDayNames[trimmed.toLowerCase()] ?? null;
  return null;
}

function isCronFieldPartValid(part: string, min: number, max: number, fieldIndex: number): boolean {
  if (part === '*' || part === '?') return true;
  const stepMatch = /^[*?]\/(\d+)$/.exec(part);
  if (stepMatch) return Number(stepMatch[1]) >= 1;
  const rangeStepMatch = /^(.+?)\/(\d+)$/.exec(part);
  if (rangeStepMatch) {
    if (Number(rangeStepMatch[2]) < 1) return false;
    part = rangeStepMatch[1]!;
  }
  const rangeMatch = /^(.+)-(.+)$/.exec(part);
  if (rangeMatch) {
    const low = cronTokenValue(rangeMatch[1]!, fieldIndex);
    const high = cronTokenValue(rangeMatch[2]!, fieldIndex);
    return low !== null && high !== null && low >= min && high <= max && low <= high;
  }
  const single = cronTokenValue(part, fieldIndex);
  return single !== null && single >= min && single <= max;
}

function splitCronTimezone(value: string): string | null {
  const match = /^(?:TZ|CRON_TZ)=([^\s]+)\s+(.+)$/.exec(value);
  if (!match) return value;
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: match[1] }).format();
  } catch {
    return null;
  }
  return match[2]!;
}

// 与 robfig-cron ParseStandard（5 字段 + 描述符）对齐。
export function isCronExpressionValid(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return false;
  const cronValue = splitCronTimezone(trimmed);
  if (!cronValue) return false;
  if (cronDescriptorSet.has(cronValue)) return true;
  if (cronValue.startsWith('@every ')) return isEveryDuration(cronValue);
  const fields = cronValue.split(/\s+/);
  if (fields.length !== 5) return false;
  return fields.every((field, index) => {
    const [min, max] = cronFieldBounds[index]!;
    return field.split(',').every(part => isCronFieldPartValid(part, min, max, index));
  });
}

export type BaseSettingFormatError = {
  key: string;
  label: string;
  message: string;
};

const rateFormatFields = [
  { key: 'personalReplenishRate', label: '个人速率' },
  { key: 'groupReplenishRate', label: '群组速率' },
];

const cronFormatFields = [{ key: 'aliveNoticeValue', label: '存活消息间隔' }];

// 仅校验保存补丁中实际修改的文本字段，未修改的字段不受影响。
export function validateBaseSettingPatchFormats(payload: Record<string, unknown>) {
  const errors: BaseSettingFormatError[] = [];
  for (const { key, label } of rateFormatFields) {
    if (!(key in payload)) continue;
    if (!isRateFormatValid(String(payload[key] ?? ''))) {
      errors.push({
        key,
        label,
        message: label + '格式无效，应为正整数或 @every 时长（如 3、@every 1s）',
      });
    }
  }
  for (const { key, label } of cronFormatFields) {
    if (!(key in payload)) continue;
    if (!isCronExpressionValid(String(payload[key] ?? ''))) {
      errors.push({
        key,
        label,
        message:
          label + '格式无效，支持 @every 时长（如 @every 3h）或标准 cron 表达式（如 0 12 * * *）',
      });
    }
  }
  return errors;
}

function normalizeExtDefaultSearchText(item: BaseSettingExtDefaultSettingItem) {
  return [item.name, ...Object.keys(item.disabledCommand ?? {})].join(' ').trim().toLowerCase();
}

function collectChangedCommands(
  current: BaseSettingExtDefaultSettingItem,
  initial?: BaseSettingExtDefaultSettingItem
) {
  const changed = new Set<string>();
  const currentCommands = current.disabledCommand ?? {};
  const initialCommands = initial?.disabledCommand ?? {};
  for (const name of Object.keys(currentCommands)) {
    if (currentCommands[name] !== initialCommands[name]) changed.add(name);
  }
  for (const name of Object.keys(initialCommands)) {
    if (initialCommands[name] !== currentCommands[name]) changed.add(name);
  }
  return [...changed].sort((left, right) => left.localeCompare(right));
}

export function buildExtDefaultSettingsView(
  currentItems: BaseSettingExtDefaultSettingItem[],
  initialItems: BaseSettingExtDefaultSettingItem[]
): ExtDefaultSettingsViewItem[] {
  const initialMap = new Map(initialItems.map(item => [item.name, item]));
  return currentItems.map((item, index) => {
    const initialItem = initialMap.get(item.name);
    const commandNames = Object.keys(item.disabledCommand ?? {}).sort((left, right) =>
      left.localeCompare(right)
    );
    const changedCommands = collectChangedCommands(item, initialItem);
    const autoActiveDirty = !initialItem || item.autoActive !== initialItem.autoActive;
    const disabledCommandDirty = changedCommands.length > 0;
    return {
      item,
      originalIndex: index,
      commandCount: commandNames.length,
      disabledCount: commandNames.filter(name => item.disabledCommand[name]).length,
      searchText: normalizeExtDefaultSearchText(item),
      dirty: autoActiveDirty || disabledCommandDirty,
      autoActiveDirty,
      disabledCommandDirty,
      changedCommands,
    };
  });
}

export function getExtDefaultSettingModifiedCount(items: ExtDefaultSettingsViewItem[]) {
  return items.filter(item => item.dirty).length;
}

export function searchExtDefaultSettingsView(items: ExtDefaultSettingsViewItem[], keyword: string) {
  const value = keyword.trim().toLowerCase();
  if (!value) return items;
  return items.filter(item => item.searchText.includes(value));
}

export function filterExtDefaultSettingsView(
  items: ExtDefaultSettingsViewItem[],
  mode: ExtDefaultSettingsFilterMode
) {
  if (mode === 'modified') return items.filter(item => item.dirty);
  return items;
}

export function sortExtDefaultSettingsView(
  items: ExtDefaultSettingsViewItem[],
  sortKey: ExtDefaultSettingsSortKey
) {
  const collator = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' });
  const sorted = [...items];
  sorted.sort((left, right) => {
    switch (sortKey) {
      case 'modified':
        if (left.dirty !== right.dirty) return left.dirty ? -1 : 1;
        break;
      case 'name': {
        const result = collator.compare(left.item.name, right.item.name);
        if (result !== 0) return result;
        break;
      }
      case 'auto-active':
        if (left.item.autoActive !== right.item.autoActive) return left.item.autoActive ? -1 : 1;
        break;
      case 'disabled-count':
        if (left.disabledCount !== right.disabledCount)
          return right.disabledCount - left.disabledCount;
        break;
      case 'source':
      default:
        break;
    }
    return left.originalIndex - right.originalIndex;
  });
  return sorted;
}

export function getExtDefaultSettingPage(
  items: ExtDefaultSettingsViewItem[],
  page: number,
  pageSize: number
) {
  const safePageSize = Math.max(1, pageSize);
  const total = items.length;
  const pageCount = Math.max(1, Math.ceil(total / safePageSize));
  const safePage = Math.min(Math.max(1, page), pageCount);
  const start = (safePage - 1) * safePageSize;
  return {
    page: safePage,
    pageCount,
    items: items.slice(start, start + safePageSize),
    total,
  };
}

export function isBaseSettingDirty(current: BaseSettingValueModel, initial: BaseSettingValueModel) {
  return JSON.stringify(current) !== JSON.stringify(initial);
}

export function buildBaseSettingPatch(
  current: BaseSettingValueModel,
  initial: BaseSettingValueModel
) {
  const changes = (object: Record<string, unknown>, base: Record<string, unknown>) =>
    transform(
      object,
      (result: Record<string, unknown>, value, key) => {
        if (isArray(value)) {
          if (!isEqual(value, base[key])) {
            result[key] = value;
          }
          return;
        }
        if (!isEqual(value, base[key])) {
          result[key] =
            isObject(value) && isObject(base[key])
              ? changes(value as Record<string, unknown>, base[key] as Record<string, unknown>)
              : value;
        }
      },
      {}
    );

  return changes(current as Record<string, unknown>, initial as Record<string, unknown>);
}
