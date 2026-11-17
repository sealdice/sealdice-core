import { parseReplyImportLine, parseReplyImportText } from './importParser';
import { expect, it } from 'vitest';

it('自定义回复导入解析合法内容与错误', async () => {
  expect(parseReplyImportLine('ping/pong')).toEqual({
    conditions: ['ping'],
    replies: ['pong'],
    rest: '',
  });

  expect(parseReplyImportLine('a|b/c|d')).toEqual({
    conditions: ['a', 'b'],
    replies: ['c', 'd'],
    rest: '',
  });

  expect(parseReplyImportLine('a/|b/c')).toEqual({
    conditions: ['a/', 'b'],
    replies: ['c'],
    rest: '',
  });

  expect(parseReplyImportLine('a/line\\nnext\nrest')).toEqual({
    conditions: ['a'],
    replies: ['line\nnext'],
    rest: 'rest',
  });

  const tasks = parseReplyImportText('a/b\nc/d');
  expect(tasks.length).toBe(2);
  expect(tasks[0]!.conditions[0]!.value).toBe('a');
  expect(tasks[1]!.results[0]!.message).toEqual([['d', 1]]);
});
