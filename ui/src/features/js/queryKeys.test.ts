import {
  jsConfigsQueryKey,
  jsDataInfoQueryKey,
  jsDataListQueryKey,
  jsDeadConfigsQueryKey,
  jsListForDataQueryKey,
  jsListQueryKey,
} from './queryKeys';
import { expect, it } from 'vitest';

it('脚本查询键隔离不同脚本与分页', async () => {
  expect(jsListQueryKey({ page: 1, pageSize: 20, sortBy: 'name', sortOrder: 'asc' })).toEqual([
    'js-list',
    { page: 1, pageSize: 20, sortBy: 'name', sortOrder: 'asc' },
  ]);
  expect(jsConfigsQueryKey()).toEqual(['js-configs']);
  expect(jsDeadConfigsQueryKey()).toEqual(['js-dead-configs']);
  expect(jsListForDataQueryKey()).toEqual(['js-list-for-data']);
  expect(jsDataListQueryKey('plugin-a', { page: 2, pageSize: 20 }, 'abc')).toEqual([
    'js-data-list',
    'plugin-a',
    { page: 2, pageSize: 20 },
    'abc',
  ]);
  expect(jsDataInfoQueryKey('plugin-a')).toEqual(['js-data-info', 'plugin-a']);
});
