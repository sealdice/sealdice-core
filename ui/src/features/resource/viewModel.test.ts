import {
  buildResourceListQuery,
  buildSealImageCode,
  createDefaultResourceListQuery,
  formatResourcePageSummary,
  formatResourceTypeLabel,
  isResourceDetailAvailable,
  isResourceUploadFileAccepted,
} from './viewModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  const normalize = (value: unknown): string => {
    if (Array.isArray(value)) {
      return `[${value.map(normalize).join(',')}]`;
    }
    if (value && typeof value === 'object') {
      const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b));
      return `{${entries.map(([key, entry]) => `${key}:${normalize(entry)}`).join(',')}}`;
    }
    return JSON.stringify(value);
  };
  if (normalize(actual) !== normalize(expected)) {
    throw new Error(`expected ${normalize(expected)}, got ${normalize(actual)}`);
  }
};

assertDeepEqual(createDefaultResourceListQuery(), {
  page: 1,
  pageSize: 20,
  keyword: '',
  type: 'image',
  sortBy: 'name',
  sortOrder: 'asc',
});

assertDeepEqual(buildResourceListQuery({
  page: 0,
  pageSize: 999,
  keyword: ' seal ',
  type: 'image',
  sortBy: 'size',
  sortOrder: 'desc',
}), {
  page: 1,
  pageSize: 20,
  keyword: 'seal',
  type: 'image',
  sortBy: 'size',
  sortOrder: 'desc',
});

assertEqual(isResourceUploadFileAccepted({ name: 'seal.PNG', type: 'image/png' }), true);
assertEqual(isResourceUploadFileAccepted({ name: 'seal.jpeg', type: 'image/jpeg' }), true);
assertEqual(isResourceUploadFileAccepted({ name: 'seal.webp', type: 'image/webp' }), false);
assertEqual(isResourceUploadFileAccepted({ name: 'seal.txt', type: 'image/png' }), false);

assertEqual(buildSealImageCode(' data/images/seal.png '), '[图:data/images/seal.png]');
assertEqual(formatResourceTypeLabel('image'), '图片');
assertEqual(formatResourceTypeLabel('unknown'), '未知');
assertEqual(formatResourcePageSummary({ total: 42, page: 2, pageSize: 20 }), '第 2 页，每页 20 个，共 42 个资源');
assertEqual(isResourceDetailAvailable({ path: 'data/images/seal.png' }), true);
assertEqual(isResourceDetailAvailable({ path: '' }), false);
