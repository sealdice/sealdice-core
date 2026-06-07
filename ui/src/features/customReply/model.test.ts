import {
  cloneReplyFileDraft,
  cloneReplyTask,
  normalizeCondition,
  normalizeReplyFileDetail,
  normalizeReplyTask,
  toApiReplyConfig,
  type ReplyFileDraft,
} from './model';

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

const task = normalizeReplyTask({
  enable: false,
  conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '12' }],
  results: [{ resultType: 'replyToSender', delay: '3', message: [['ok', '2']] }],
});

assertEqual(task.enable, false);
assertDeepEqual(task.conditions, [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '12' }]);
assertDeepEqual(task.results[0]?.message, [['ok', 2]]);

const numericCondition = normalizeCondition({ condType: 'textLenLimit', matchType: 'matchGreater', value: 5 });
assertEqual(numericCondition.value, 5);

const draft = normalizeReplyFileDetail({
  enable: true,
  interval: 2,
  name: 'demo',
  author: ['a'],
  version: '1',
  createTimestamp: 1,
  updateTimestamp: 2,
  desc: 'desc',
  storeID: 'store',
  conditions: [{ condType: 'textMatch', matchType: 'matchExact', value: 'ping' }],
  filename: 'demo.yaml',
  itemCount: 1,
} as never);

assertEqual(draft.filename, 'demo.yaml');
assertEqual(draft.items.length, 0);

const clonedTask = cloneReplyTask(task);
clonedTask.results[0]!.message[0]![0] = 'changed';
assertEqual(task.results[0]!.message[0]![0], 'ok');

const clonedDraft = cloneReplyFileDraft({
  ...draft,
  items: [task],
} satisfies ReplyFileDraft);
clonedDraft.author.push('b');
assertDeepEqual(draft.author, ['a']);

const payload = toApiReplyConfig({
  ...draft,
  items: [task],
  conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '7' }],
});
assertEqual(payload.conditions[0]!.value, 7);
assertEqual(payload.items[0]!.results[0]!.delay, 3);
