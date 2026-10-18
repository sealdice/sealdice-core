import { expect, it } from 'vitest';
import type { BaseSettingValueResp } from '@/api';
import { useBaseSettingDraft } from './draft';

function createRemoteValue() {
  return {
    commandPrefix: ['.'],
    diceMasters: ['QQ:1'],
    noticeIds: [],
    extDefaultSettings: [{ name: 'core', autoActive: true, disabledCommand: { help: false } }],
    serveAddress: '127.0.0.1:3211',
    uiPassword: '------',
  } as unknown as BaseSettingValueResp;
}

// 草稿存在 ref 里，读出来的值是响应式代理；updateField 的 spread 还会把嵌套层
// 留成代理。structuredClone 对代理会抛 DataCloneError，一旦抛出，保存与放弃
// 改动都会静默失败，页面看起来像按钮没反应。
it('放弃改动能把代理草稿还原回远端值', () => {
  const draft = useBaseSettingDraft();
  draft.syncRemote(createRemoteValue());

  draft.currentValue.value = { ...draft.currentValue.value!, serveAddress: '127.0.0.1:4000' };
  expect(draft.dirty.value).toBe(true);

  draft.resetToRemote();

  expect(draft.dirty.value).toBe(false);
  expect(draft.currentValue.value?.serveAddress).toBe('127.0.0.1:3211');
});

it('保存后提交基线能清掉脏态', () => {
  const draft = useBaseSettingDraft();
  draft.syncRemote(createRemoteValue());

  draft.currentValue.value = { ...draft.currentValue.value!, serveAddress: '127.0.0.1:5000' };
  draft.commitSaved();

  expect(draft.dirty.value).toBe(false);
  expect(draft.initialValue.value?.serveAddress).toBe('127.0.0.1:5000');
});

it('嵌套层被改写时基线不跟着变', () => {
  const draft = useBaseSettingDraft();
  draft.syncRemote(createRemoteValue());

  draft.currentValue.value!.extDefaultSettings[0]!.autoActive = false;

  expect(draft.dirty.value).toBe(true);
  expect(draft.initialValue.value?.extDefaultSettings[0]?.autoActive).toBe(true);
});
