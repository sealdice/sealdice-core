import {
  buildResourceListQuery,
  buildSealImageCode,
  createDefaultResourceListQuery,
  formatResourceTypeLabel,
  isResourceDetailAvailable,
  isResourceUploadFileAccepted,
} from './viewModel.js';
import { expect, it } from 'vitest';

it('资源列表数据与筛选项转换', async () => {
  expect(createDefaultResourceListQuery()).toEqual({
    page: 1,
    pageSize: 20,
    keyword: '',
    type: 'image',
    sortBy: 'name',
    sortOrder: 'asc',
  });

  expect(
    buildResourceListQuery({
      page: 0,
      pageSize: 999,
      keyword: ' seal ',
      type: 'image',
      sortBy: 'size',
      sortOrder: 'desc',
    })
  ).toEqual({
    page: 1,
    pageSize: 20,
    keyword: 'seal',
    type: 'image',
    sortBy: 'size',
    sortOrder: 'desc',
  });

  expect(isResourceUploadFileAccepted({ name: 'seal.PNG', type: 'image/png' })).toBe(true);
  expect(isResourceUploadFileAccepted({ name: 'seal.jpeg', type: 'image/jpeg' })).toBe(true);
  expect(isResourceUploadFileAccepted({ name: 'seal.webp', type: 'image/webp' })).toBe(false);
  expect(isResourceUploadFileAccepted({ name: 'seal.txt', type: 'image/png' })).toBe(false);

  expect(buildSealImageCode(' data/images/seal.png ')).toBe('[图:data/images/seal.png]');
  expect(formatResourceTypeLabel('image')).toBe('图片');
  expect(formatResourceTypeLabel('unknown')).toBe('未知');
  expect(isResourceDetailAvailable({ path: 'data/images/seal.png' })).toBe(true);
  expect(isResourceDetailAvailable({ path: '' })).toBe(false);
});
