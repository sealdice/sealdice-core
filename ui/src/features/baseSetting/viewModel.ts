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
export type ExtDefaultSettingsSortKey = 'source' | 'modified' | 'name' | 'auto-active' | 'disabled-count';

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

export function cloneBaseSettingValue(value: BaseSettingValueModel): BaseSettingValueModel {
  return structuredClone(value);
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

export function buildBaseSettingSearchIndex(schema: BaseSettingSchemaModel): BaseSettingSearchEntry[] {
  const entries: BaseSettingSearchEntry[] = [];
  for (const tab of schema.tabs) {
    for (const group of tab.groups) {
      for (const field of group.fields) {
        const tokens = [
          tab.title,
          group.title,
          field.label,
          field.hint,
          ...(field.keywords ?? []),
        ]
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

export function getBaseSettingFieldLayout(field: Pick<BaseSettingFieldModel, 'kind'>): BaseSettingFieldLayout {
  if (['ext-default-settings', 'string-list', 'upload'].includes(field.kind)) return 'stacked';
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

function normalizeExtDefaultSearchText(item: BaseSettingExtDefaultSettingItem) {
  return [item.name, ...Object.keys(item.disabledCommand ?? {})]
    .join(' ')
    .trim()
    .toLowerCase();
}

function collectChangedCommands(
  current: BaseSettingExtDefaultSettingItem,
  initial?: BaseSettingExtDefaultSettingItem,
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
  initialItems: BaseSettingExtDefaultSettingItem[],
): ExtDefaultSettingsViewItem[] {
  const initialMap = new Map(initialItems.map(item => [item.name, item]));
  return currentItems.map((item, index) => {
    const initialItem = initialMap.get(item.name);
    const commandNames = Object.keys(item.disabledCommand ?? {}).sort((left, right) => left.localeCompare(right));
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
  mode: ExtDefaultSettingsFilterMode,
) {
  if (mode === 'modified') return items.filter(item => item.dirty);
  return items;
}

export function sortExtDefaultSettingsView(
  items: ExtDefaultSettingsViewItem[],
  sortKey: ExtDefaultSettingsSortKey,
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
        if (left.disabledCount !== right.disabledCount) return right.disabledCount - left.disabledCount;
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
  pageSize: number,
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

export function buildBaseSettingPatch(current: BaseSettingValueModel, initial: BaseSettingValueModel) {
  const changes = (object: Record<string, unknown>, base: Record<string, unknown>) =>
    transform(object, (result: Record<string, unknown>, value, key) => {
      if (isArray(value)) {
        if (!isEqual(value, base[key])) {
          result[key] = value;
        }
        return;
      }
      if (!isEqual(value, base[key])) {
        result[key] = isObject(value) && isObject(base[key])
          ? changes(value as Record<string, unknown>, base[key] as Record<string, unknown>)
          : value;
      }
    }, {});

  return changes(current as Record<string, unknown>, initial as Record<string, unknown>);
}
