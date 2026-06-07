import { pinyin } from 'pinyin-pro';
import type { NavigationBreadcrumbItem, NavigationItem, NavigationSearchItem } from './types';

export interface BuildNavigationOptions {
  advancedConfigEnabled: boolean;
  customTextCategories: string[];
}

export function buildNavigationTree(
  sourceItems: NavigationItem[],
  options: BuildNavigationOptions,
): NavigationItem[] {
  return sourceItems
    .map(item => buildNavigationItem(item, options))
    .filter((item): item is NavigationItem => Boolean(item));
}

function buildNavigationItem(
  item: NavigationItem,
  options: BuildNavigationOptions,
): NavigationItem | undefined {
  if (item.hidden) return undefined;
  if (item.requiresAdvancedConfig && !options.advancedConfigEnabled) return undefined;

  const children = item.dynamicChildren === 'customTextCategories'
    ? options.customTextCategories.map(category => ({
        label: category,
        path: `/custom-text/${category}`,
        icon: 'dice',
        title: category,
        layout: item.layout,
      }))
    : item.children
      ?.map(child => buildNavigationItem(child, options))
      .filter((child): child is NavigationItem => Boolean(child));

  return {
    ...item,
    ...(children ? { children } : {}),
  };
}

export function buildRouteMeta(items: NavigationItem[]) {
  const meta: Record<string, { title?: string; layout?: NavigationItem['layout'] }> = {};

  const walk = (item: NavigationItem) => {
    if (item.hidden) return;

    const title = item.title ?? item.label;
    if (item.path) {
      meta[item.path] = {
        title,
        ...(item.layout ? { layout: item.layout } : {}),
      };
    }

    if (item.dynamicChildren === 'customTextCategories') {
      meta['/custom-text/:category'] = {
        title,
        ...(item.layout ? { layout: item.layout } : {}),
      };
    }

    item.children?.forEach(walk);
  };

  items.forEach(walk);
  return meta;
}

export function buildBreadcrumbItems(
  items: NavigationItem[],
  activePath: string,
  fallbackTitle = '当前页面',
): NavigationBreadcrumbItem[] {
  const path = activePath === '/home' ? '/' : normalizePath(activePath);

  const staticMatch = findStaticBreadcrumb(items, path, []);
  if (staticMatch) return staticMatch;

  const dynamicMatch = findDynamicBreadcrumb(items, path, []);
  if (dynamicMatch) return dynamicMatch;

  return [{ label: fallbackTitle }];
}

function findStaticBreadcrumb(
  items: NavigationItem[],
  activePath: string,
  parents: NavigationBreadcrumbItem[],
): NavigationBreadcrumbItem[] | undefined {
  for (const item of items) {
    if (item.hidden) continue;

    const label = item.title ?? item.label;
    if (item.path && normalizePath(item.path) === activePath) {
      return [...parents, { label }];
    }

    const nextParents = item.path ? parents : [...parents, { label }];
    const childMatch = findStaticBreadcrumb(item.children ?? [], activePath, nextParents);
    if (childMatch) return childMatch;
  }

  return undefined;
}

function findDynamicBreadcrumb(
  items: NavigationItem[],
  activePath: string,
  parents: NavigationBreadcrumbItem[],
): NavigationBreadcrumbItem[] | undefined {
  for (const item of items) {
    if (item.hidden) continue;

    const label = item.title ?? item.label;
    if (item.dynamicChildren === 'customTextCategories' && activePath.startsWith('/custom-text/')) {
      const category = activePath.split('/').slice(2).join('/');
      return [...parents, { label }, { label: decodePathPart(category) || '当前页面' }];
    }

    const nextParents = item.path ? parents : [...parents, { label }];
    const childMatch = findDynamicBreadcrumb(item.children ?? [], activePath, nextParents);
    if (childMatch) return childMatch;
  }

  return undefined;
}

export function flattenNavigationItems(items: NavigationItem[]): NavigationSearchItem[] {
  const result: NavigationSearchItem[] = [];

  const walk = (item: NavigationItem, inheritedIcon?: string) => {
    const icon = item.icon ?? inheritedIcon;
    if (item.path && !item.children?.length) {
      result.push({
        label: item.label,
        path: item.path,
        ...(icon ? { icon } : {}),
      });
    }

    item.children?.forEach(child => walk(child, icon));
  };

  items.forEach(item => walk(item));
  return result;
}

export function getNavigationExpandedKeys(items: NavigationItem[], activePath: string): string[] {
  const normalizedActivePath = normalizePath(activePath);

  const walk = (item: NavigationItem, parents: string[]): string[] | undefined => {
    if (item.path && normalizePath(item.path) === normalizedActivePath) return parents;

    for (const child of item.children ?? []) {
      const result = walk(child, [...parents, item.path ?? item.label]);
      if (result) return result;
    }

    return undefined;
  };

  for (const item of items) {
    const result = walk(item, []);
    if (result) return result;
  }

  return [];
}

export function matchesNavigationSearch(item: NavigationSearchItem, query: string): boolean {
  const normalizedQuery = normalize(query);
  if (!normalizedQuery) return false;

  return [
    item.label,
    item.path,
    toPinyin(item.label),
    toPinyinInitials(item.label),
  ].some(value => normalize(value).includes(normalizedQuery));
}

export function addSearchHistory(
  history: NavigationSearchItem[],
  item: NavigationSearchItem,
): NavigationSearchItem[] {
  return [item, ...history.filter(historyItem => historyItem.path !== item.path)].slice(0, 10);
}

export function removeSearchHistoryItem(
  history: NavigationSearchItem[],
  path: string,
): NavigationSearchItem[] {
  return history.filter(item => item.path !== path);
}

function normalize(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, '');
}

function normalizePath(path: string): string {
  try {
    return decodeURIComponent(path);
  } catch {
    return path;
  }
}

function decodePathPart(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function toPinyin(value: string): string {
  return pinyin(value, {
    toneType: 'none',
    type: 'array',
  }).join('');
}

function toPinyinInitials(value: string): string {
  return pinyin(value, {
    pattern: 'first',
    toneType: 'none',
    type: 'array',
  }).join('');
}
