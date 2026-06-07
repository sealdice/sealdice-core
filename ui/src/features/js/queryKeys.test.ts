import {
  jsConfigsQueryKey,
  jsDataInfoQueryKey,
  jsDataListQueryKey,
  jsDeadConfigsQueryKey,
  jsListForDataQueryKey,
  jsListQueryKey,
} from './queryKeys';

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(jsListQueryKey({ page: 1, pageSize: 20, sortBy: 'name', sortOrder: 'asc' }), [
  'js-list',
  { page: 1, pageSize: 20, sortBy: 'name', sortOrder: 'asc' },
]);
assertDeepEqual(jsConfigsQueryKey(), ['js-configs']);
assertDeepEqual(jsDeadConfigsQueryKey(), ['js-dead-configs']);
assertDeepEqual(jsListForDataQueryKey(), ['js-list-for-data']);
assertDeepEqual(jsDataListQueryKey('plugin-a', { page: 2, pageSize: 20 }, 'abc'), [
  'js-data-list',
  'plugin-a',
  { page: 2, pageSize: 20 },
  'abc',
]);
assertDeepEqual(jsDataInfoQueryKey('plugin-a'), ['js-data-info', 'plugin-a']);
