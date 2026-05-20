import { trim } from 'es-toolkit/compat';
import type { HelpDoc, HelpTextVo } from '@/api';

export type GroupOption = {
  label: string;
  value: string;
};

export type HelpDocTreeOption = {
  key: string;
  label: string;
  raw: HelpDoc;
  disabled?: boolean;
  children?: HelpDocTreeOption[];
  icon: 'folder' | 'json' | 'xlsx' | 'file';
  tag?: ReturnType<typeof getHelpdocTag>;
};

export function normalizeHelpdocAliases(aliases: Record<string, string[] | null | undefined>) {
  const result = new Map<string, string[]>();
  for (const [key, value] of Object.entries(aliases)) {
    result.set(key, Array.isArray(value) ? [...value] : []);
  }
  return result;
}

export function buildHelpdocGroupOptions(tree: HelpDoc[] | undefined) {
  const groups: GroupOption[] = [{ label: '默认', value: 'default' }];
  for (const entry of tree ?? []) {
    if (entry.isDir && entry.name) {
      groups.push({ label: entry.name, value: entry.name });
    }
  }
  return groups;
}

export function getHelpdocTag(loadStatus: number, deleted: boolean, group: string) {
  if (loadStatus === 0) return { type: 'warning' as const, label: '未加载' };
  if (loadStatus === 2) return { type: 'error' as const, label: '格式有误' };
  if (deleted) return { type: 'warning' as const, label: group || 'default' };
  return { type: 'success' as const, label: group || 'default' };
}

export function convertHelpdocTree(doc: HelpDoc): HelpDocTreeOption {
  return {
    key: doc.key,
    label: doc.name,
    raw: doc,
    disabled: doc.deleted,
    children: doc.children?.map(convertHelpdocTree) ?? undefined,
    icon: doc.isDir ? 'folder' : doc.type === '.json' ? 'json' : doc.type === '.xlsx' ? 'xlsx' : 'file',
    tag: doc.isDir
      ? undefined
      : getHelpdocTag(doc.loadStatus ?? 0, Boolean(doc.deleted), doc.group ?? ''),
  };
}

export function getHelpdocTextPreview(row: string) {
  const text = trim(row);
  if (text.length <= 200) return text;
  return `${text.slice(0, 151)}...`;
}

export function getHelpdocTextTooltip(row: string) {
  return trim(row);
}

export function buildHelpdocItemParams(query: {
  pageNum: number;
  pageSize: number;
  id: number | null;
  group: string | null;
  from: string;
  title: string;
}) {
  return {
    pageNum: query.pageNum,
    pageSize: query.pageSize,
    id: query.id ? String(query.id) : undefined,
    group: query.group || undefined,
    from: query.from || undefined,
    title: query.title || undefined,
  };
}

export function isHelpdocUploadFileAccepted(filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase();
  return ext === 'json' || ext === 'xlsx';
}

export function getHelpdocItemGroupOptions(tree: HelpDoc[] | undefined) {
  return [{ label: '内置', value: 'builtin' }, ...buildHelpdocGroupOptions(tree)];
}

export function isHelpdocConfigDirty(current: Map<string, string[]>, initial: Map<string, string[]>) {
  if (current.size !== initial.size) return true;
  for (const [key, value] of current) {
    const other = initial.get(key) ?? [];
    if (value.length !== other.length) return true;
    for (let index = 0; index < value.length; index += 1) {
      if (value[index] !== other[index]) return true;
    }
  }
  return false;
}

export function buildHelpdocConfigPayload(aliases: Map<string, string[]>) {
  return Object.fromEntries(aliases);
}

export function cloneHelpdocAliases(source: Map<string, string[]>) {
  return new Map(Array.from(source.entries(), ([key, value]) => [key, [...value]]));
}

export type HelpdocItemRow = Pick<HelpTextVo, 'id' | 'group' | 'from' | 'title' | 'content' | 'packageName'>;
