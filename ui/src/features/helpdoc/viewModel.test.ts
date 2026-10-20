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
  summarizeHelpdocDelete,
} from './viewModel.js';
import type { HelpDoc } from '@/api';
import { expect, it } from 'vitest';

it('帮助文档树与分页模型转换', async () => {
  const tree: HelpDoc[] = [
    {
      key: 'default',
      name: 'default',
      group: '',
      path: 'default',
      type: '',
      loadStatus: 1,
      deleted: false,
      isDir: true,
      children: [
        {
          key: 'default/test.json',
          name: 'test.json',
          group: 'default',
          path: 'default/test.json',
          type: '.json',
          deleted: false,
          isDir: false,
          loadStatus: 1,
          children: [],
        },
      ],
    },
  ];

  expect(buildHelpdocGroupOptions(tree)).toEqual([
    { label: '默认', value: 'default' },
    { label: 'default', value: 'default' },
  ]);

  const converted = convertHelpdocTree(tree[0]);
  expect(converted.icon).toBe('folder');
  expect(converted.children?.[0]?.icon).toBe('json');
  expect(converted.children?.[0]?.tag).toEqual({ type: 'success', label: 'default' });
  expect(summarizeHelpdocDelete([converted], ['default/test.json'])).toEqual({
    count: 1,
    labels: ['test.json'],
  });

  expect(getHelpdocTag(0, false, 'default')).toEqual({ type: 'warning', label: '未加载' });
  expect(getHelpdocTag(2, false, 'default')).toEqual({ type: 'error', label: '格式有误' });
  expect(getHelpdocTag(1, true, 'group-a')).toEqual({ type: 'warning', label: 'group-a' });

  expect(getHelpdocTextPreview('  short text  ')).toBe('short text');
  expect(getHelpdocTextPreview(` ${'a'.repeat(205)} `)).toBe(`${'a'.repeat(151)}...`);

  expect(isHelpdocUploadFileAccepted('a.json')).toBe(true);
  expect(isHelpdocUploadFileAccepted('a.xlsx')).toBe(true);
  expect(isHelpdocUploadFileAccepted('a.txt')).toBe(false);

  expect(
    buildHelpdocItemParams({
      pageNum: 2,
      pageSize: 30,
      id: 9,
      group: 'builtin',
      from: 'foo',
      title: 'bar',
    })
  ).toEqual({
    pageNum: 2,
    pageSize: 30,
    id: '9',
    group: 'builtin',
    from: 'foo',
    title: 'bar',
  });

  const aliases = normalizeHelpdocAliases({
    default: ['基础'],
    empty: null,
  });
  expect(Array.from(aliases.entries())).toEqual([
    ['default', ['基础']],
    ['empty', []],
  ]);

  const cloned = cloneHelpdocAliases(aliases);
  cloned.get('default')?.push('同义词');
  expect(aliases.get('default')).toEqual(['基础']);
  expect(
    isHelpdocConfigDirty(aliases, normalizeHelpdocAliases({ default: ['基础'], empty: null }))
  ).toBe(false);
  expect(isHelpdocConfigDirty(cloned, aliases)).toBe(true);
});
