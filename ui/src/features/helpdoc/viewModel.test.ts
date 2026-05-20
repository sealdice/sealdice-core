import {
  buildHelpdocGroupOptions,
  buildHelpdocItemParams,
  cloneHelpdocAliases,
  convertHelpdocTree,
  getHelpdocTag,
  getHelpdocTextPreview,
  isHelpdocConfigDirty,
  isHelpdocUploadFileAccepted,
  normalizeHelpdocAliases,
} from './viewModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const tree = [
  {
    key: 'default',
    name: 'default',
    group: '',
    type: '',
    deleted: false,
    isDir: true,
    children: [
      {
        key: 'default/test.json',
        name: 'test.json',
        group: 'default',
        type: '.json',
        deleted: false,
        isDir: false,
        loadStatus: 1,
        children: [],
      },
    ],
  },
] as any;

assertDeepEqual(buildHelpdocGroupOptions(tree), [
  { label: '默认', value: 'default' },
  { label: 'default', value: 'default' },
]);

const converted = convertHelpdocTree(tree[0]);
assertEqual(converted.icon, 'folder');
assertEqual(converted.children?.[0]?.icon, 'json');
assertDeepEqual(converted.children?.[0]?.tag, { type: 'success', label: 'default' });

assertDeepEqual(getHelpdocTag(0, false, 'default'), { type: 'warning', label: '未加载' });
assertDeepEqual(getHelpdocTag(2, false, 'default'), { type: 'error', label: '格式有误' });
assertDeepEqual(getHelpdocTag(1, true, 'group-a'), { type: 'warning', label: 'group-a' });

assertEqual(getHelpdocTextPreview('  short text  '), 'short text');
assertEqual(getHelpdocTextPreview(` ${'a'.repeat(205)} `), `${'a'.repeat(151)}...`);

assertEqual(isHelpdocUploadFileAccepted('a.json'), true);
assertEqual(isHelpdocUploadFileAccepted('a.xlsx'), true);
assertEqual(isHelpdocUploadFileAccepted('a.txt'), false);

assertDeepEqual(
  buildHelpdocItemParams({
    pageNum: 2,
    pageSize: 30,
    id: 9,
    group: 'builtin',
    from: 'foo',
    title: 'bar',
  }),
  {
    pageNum: 2,
    pageSize: 30,
    id: '9',
    group: 'builtin',
    from: 'foo',
    title: 'bar',
  },
);

const aliases = normalizeHelpdocAliases({
  default: ['基础'],
  empty: null,
});
assertDeepEqual(Array.from(aliases.entries()), [
  ['default', ['基础']],
  ['empty', []],
]);

const cloned = cloneHelpdocAliases(aliases);
cloned.get('default')?.push('同义词');
assertDeepEqual(aliases.get('default'), ['基础']);
assertEqual(isHelpdocConfigDirty(aliases, normalizeHelpdocAliases({ default: ['基础'], empty: null })), false);
assertEqual(isHelpdocConfigDirty(cloned, aliases), true);
