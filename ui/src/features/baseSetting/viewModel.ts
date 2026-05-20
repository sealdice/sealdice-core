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

export function normalizeBaseSettingValue(value: BaseSettingValueResp): BaseSettingValueModel {
  return {
    ...value,
    commandPrefix: [...(value.commandPrefix ?? [])],
    diceMasters: [...(value.diceMasters ?? [])],
    noticeIds: [...(value.noticeIds ?? [])],
    extDefaultSettings: (value.extDefaultSettings ?? []).map(item => ({
      ...item,
      disabledCommand: { ...(item.disabledCommand ?? {}) },
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
