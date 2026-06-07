import type { ResourceItem } from '@/api';

export type ResourceTypeKey = 'image';
export type ResourceListSortKey = 'name' | 'size' | 'ext' | 'path';
export type ResourceSortOrder = 'asc' | 'desc';

export type ResourceListQueryModel = {
  page: number;
  pageSize: number;
  keyword: string;
  type: ResourceTypeKey;
  sortBy: ResourceListSortKey;
  sortOrder: ResourceSortOrder;
};

export type ResourceUploadCandidate = {
  name: string;
  type?: string;
};

export const RESOURCE_PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const;

const allowedImageExts = new Set(['.png', '.jpg', '.jpeg', '.gif']);
const allowedImageMimeTypes = new Set(['', 'image/png', 'image/jpeg', 'image/gif']);

export function createDefaultResourceListQuery(): ResourceListQueryModel {
  return {
    page: 1,
    pageSize: 20,
    keyword: '',
    type: 'image',
    sortBy: 'name',
    sortOrder: 'asc',
  };
}

export function buildResourceListQuery(query: ResourceListQueryModel) {
  return {
    page: normalizePage(query.page),
    pageSize: normalizePageSize(query.pageSize),
    keyword: query.keyword.trim(),
    type: query.type,
    sortBy: query.sortBy,
    sortOrder: query.sortOrder,
  };
}

export function isResourceUploadFileAccepted(file: ResourceUploadCandidate): boolean {
  const ext = getLowerExt(file.name);
  const mimeType = (file.type ?? '').toLowerCase();
  return allowedImageExts.has(ext) && allowedImageMimeTypes.has(mimeType);
}

export function buildSealImageCode(path: string): string {
  return `[图:${path.trim()}]`;
}

export function formatResourceTypeLabel(type: string): string {
  if (type === 'image') return '图片';
  if (type === 'audio') return '音频';
  if (type === 'video') return '视频';
  return '未知';
}

export function getResourceTypeTagType(type: string) {
  if (type === 'image') return 'info' as const;
  if (type === 'audio') return 'success' as const;
  if (type === 'video') return 'warning' as const;
  return 'default' as const;
}

export function formatResourcePageSummary(page: Pick<ResourceListQueryModel, 'page' | 'pageSize'> & { total: number }): string {
  return `第 ${normalizePage(page.page)} 页，每页 ${normalizePageSize(page.pageSize)} 个，共 ${Math.max(0, page.total)} 个资源`;
}

export function getResourceKey(item: ResourceItem): string {
  return item.path || item.name;
}

export function isResourceDetailAvailable(item: Pick<ResourceItem, 'path'>): boolean {
  return item.path.trim().length > 0;
}

function normalizePage(page: number): number {
  if (!Number.isFinite(page) || page < 1) return 1;
  return Math.trunc(page);
}

function normalizePageSize(pageSize: number): number {
  if (!RESOURCE_PAGE_SIZE_OPTIONS.includes(pageSize as (typeof RESOURCE_PAGE_SIZE_OPTIONS)[number])) return 20;
  return pageSize;
}

function getLowerExt(filename: string): string {
  const index = filename.lastIndexOf('.');
  if (index < 0) return '';
  return filename.slice(index).toLowerCase();
}
