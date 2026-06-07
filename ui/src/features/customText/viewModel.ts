import { normalizeTextDict, type TextTemplateHelpDict, type TextTemplateHelpGroup, type TextTemplateItem, type TextTemplateWithWeightDict } from './types';

export type CustomTextFilterMode = 'all' | 'unmodified' | 'modified' | 'group' | 'deprecated';

export type SortCustomTextCategoryInput = {
  texts: TextTemplateWithWeightDict;
  helpInfo: TextTemplateHelpDict;
  category: string;
  filterMode: CustomTextFilterMode;
  filterName?: string;
  filterGroup?: string;
};

export type SortedCustomTextCategory = Array<[string, Array<[string, TextTemplateItem[]]>]>;

export function getCustomTextGroups(helpGroup: TextTemplateHelpGroup = {}): string[] {
  return Array.from(
    new Set(
      Object.values(helpGroup)
        .map(info => firstGroupName(info.subType))
        .filter(group => group !== ''),
    ),
  ).sort((a, b) => a.localeCompare(b));
}

export function sortCustomTextCategory(input: SortCustomTextCategoryInput): SortedCustomTextCategory {
  const categoryHelpInfo = input.helpInfo[input.category] ?? {};
  let items = Object.entries(input.texts[input.category] ?? {});
  const filterName = input.filterName ?? '';

  if (filterName !== '') {
    items = items.filter(([keyName]) => {
      const itemHelp = categoryHelpInfo[keyName];
      return keyName.includes(filterName) || itemHelp?.subType?.includes(filterName);
    });
  }

  switch (input.filterMode) {
    case 'all':
      break;
    case 'unmodified':
      items = items.filter(([keyName]) => !categoryHelpInfo[keyName]?.modified);
      break;
    case 'modified':
      items = items.filter(([keyName]) => categoryHelpInfo[keyName]?.modified);
      break;
    case 'deprecated':
      items = items.filter(([keyName]) => categoryHelpInfo[keyName]?.notBuiltin);
      break;
    case 'group': {
      const targetGroup = input.filterGroup ?? '';
      items = items.filter(([keyName]) => firstGroupName(categoryHelpInfo[keyName]?.subType).startsWith(targetGroup));
      break;
    }
  }

  const groups = new Map<string, Array<[string, TextTemplateItem[]]>>();
  const groupCounts = countGroups(items, categoryHelpInfo);
  for (const item of items) {
    const group = firstGroupName(categoryHelpInfo[item[0]]?.subType);
    const boxedGroup = group && (groupCounts.get(group) ?? 0) >= 4 ? group : '__others__';
    const groupItems = groups.get(boxedGroup) ?? [];
    groupItems.push(item);
    groups.set(boxedGroup, groupItems);
  }

  return Array.from(groups.entries())
    .map(([group, groupItems]) => [
      group,
      groupItems.sort((a, b) => compareTextItems(a[0], b[0], categoryHelpInfo)),
    ] as [string, Array<[string, TextTemplateItem[]]>])
    .sort(([aGroup], [bGroup]) => {
      if (aGroup === '__others__') return -1;
      if (bGroup === '__others__') return 1;
      return aGroup.localeCompare(bGroup);
    });
}

export function buildCustomTextExportContent(input: {
  texts: TextTemplateWithWeightDict;
  category: string;
  onlyCurrent: boolean;
  compact: boolean;
}): string {
  const indent = input.compact ? 0 : 2;
  if (input.onlyCurrent) {
    return JSON.stringify(
      {
        title: '某人的自定义配置',
        items: {
          [input.category]: input.texts[input.category],
        },
      },
      null,
      indent,
    );
  }
  return JSON.stringify(input.texts, null, indent);
}

export function parseCustomTextImportContent(content: string): TextTemplateWithWeightDict {
  const data = JSON.parse(content) as {
    title?: string;
    items?: TextTemplateWithWeightDict;
  };
  if (!(data.title && data.items)) {
    throw new Error('invalid custom text import content');
  }
  return normalizeTextDict(data.items);
}

export function createTextItemKeyStore() {
  const textItemKeys = new WeakMap<TextTemplateItem, string>();
  let nextTextItemKey = 0;
  return {
    keyOf: (keyName: string, item: TextTemplateItem): string => {
      const existing = textItemKeys.get(item);
      if (existing) return `${keyName}:${existing}`;
      nextTextItemKey += 1;
      const key = `text-item-${nextTextItemKey}`;
      textItemKeys.set(item, key);
      return `${keyName}:${key}`;
    },
  };
}

function countGroups(items: Array<[string, TextTemplateItem[]]>, helpGroup: TextTemplateHelpGroup) {
  const counts = new Map<string, number>();
  for (const [keyName] of items) {
    const group = firstGroupName(helpGroup[keyName]?.subType);
    if (!group) continue;
    counts.set(group, (counts.get(group) ?? 0) + 1);
  }
  return counts;
}

function compareTextItems(aKey: string, bKey: string, helpGroup: TextTemplateHelpGroup): number {
  const itemA = helpGroup[aKey];
  const itemB = helpGroup[bKey];

  if ((itemA?.topOrder ?? 0) !== (itemB?.topOrder ?? 0)) {
    return (itemB?.topOrder ?? 0) - (itemA?.topOrder ?? 0);
  }

  if ((itemA?.subType ?? '') !== (itemB?.subType ?? '')) {
    return (itemB?.subType ?? '').localeCompare(itemA?.subType ?? '');
  }

  return 0;
}

function firstGroupName(subType?: string): string {
  return String(subType ?? '').trim().split(' ')[0] ?? '';
}
