import { parseReplyImportLine, parseReplyImportText } from './importParser';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(parseReplyImportLine('ping/pong'), {
  conditions: ['ping'],
  replies: ['pong'],
  rest: '',
});

assertDeepEqual(parseReplyImportLine('a|b/c|d'), {
  conditions: ['a', 'b'],
  replies: ['c', 'd'],
  rest: '',
});

assertDeepEqual(parseReplyImportLine('a/|b/c'), {
  conditions: ['a/', 'b'],
  replies: ['c'],
  rest: '',
});

assertDeepEqual(parseReplyImportLine('a/line\\nnext\nrest'), {
  conditions: ['a'],
  replies: ['line\nnext'],
  rest: 'rest',
});

const tasks = parseReplyImportText('a/b\nc/d');
assertEqual(tasks.length, 2);
assertEqual(tasks[0]!.conditions[0]!.value, 'a');
assertDeepEqual(tasks[1]!.results[0]!.message, [['d', 1]]);
