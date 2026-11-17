import {
  cloneReplyFileDraft,
  cloneReplyTask,
  normalizeCondition,
  normalizeReplyFileDetail,
  normalizeReplyTask,
  toApiReplyConfig,
  type ReplyFileDraft,
} from './model';
import { expect, it } from 'vitest';

it('回复草稿规范化及 API 数据转换', async () => {
  const task = normalizeReplyTask({
    enable: false,
    conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '12' }],
    results: [{ resultType: 'replyToSender', delay: '3', message: [['ok', '2']] }],
  });

  expect(task.enable).toBe(false);
  expect(task.conditions).toEqual([
    { condType: 'textLenLimit', matchType: 'matchGreater', value: '12' },
  ]);
  expect(task.results[0]?.message).toEqual([['ok', 2]]);

  const numericCondition = normalizeCondition({
    condType: 'textLenLimit',
    matchType: 'matchGreater',
    value: 5,
  });
  expect(numericCondition.value).toBe(5);

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
    packageId: 'author/reply-pack',
    conditions: [{ condType: 'textMatch', matchType: 'matchExact', value: 'ping' }],
    filename: 'demo.yaml',
    itemCount: 1,
  } as never);

  expect(draft.filename).toBe('demo.yaml');
  expect(draft.items.length).toBe(0);

  const clonedTask = cloneReplyTask(task);
  clonedTask.results[0]!.message[0]![0] = 'changed';
  expect(task.results[0]!.message[0]![0]).toBe('ok');

  const clonedDraft = cloneReplyFileDraft({
    ...draft,
    items: [task],
  } satisfies ReplyFileDraft);
  clonedDraft.author.push('b');
  expect(draft.author).toEqual(['a']);

  const payload = toApiReplyConfig({
    ...draft,
    items: [task],
    conditions: [{ condType: 'textLenLimit', matchType: 'matchGreater', value: '7' }],
  });
  expect(payload.conditions[0]!.value).toBe(7);
  expect(payload.packageId).toBe('author/reply-pack');
  expect(payload.items[0]!.results[0]!.delay).toBe(3);
});
