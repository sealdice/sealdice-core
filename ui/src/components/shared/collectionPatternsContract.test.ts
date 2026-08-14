import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

describe('editable collection contract', () => {
  it('keeps add actions below the items and uses the shared primary-secondary action', () => {
    const source = readComponent('RepeatableList.vue');

    expect(source.indexOf('sd-repeatable-list__items')).toBeLessThan(
      source.indexOf('sd-repeatable-list__footer')
    );
    expect(source).toMatch(/type="primary"/);
    expect(source).toMatch(/secondary/);
    expect(source).toMatch(/i-tabler-plus/);
  });

  it('keeps enable and destructive controls in the item header', () => {
    const source = readComponent('RepeatableItem.vue');

    expect(source).toMatch(/sd-repeatable-item__header/);
    expect(source).toMatch(/<n-switch/);
    expect(source).toMatch(/type="error"/);
    expect(source).toMatch(/i-tabler-trash/);
    expect(source).toMatch(/aria-label="removeLabel"/);
  });
});

describe('repeatable setting migrations', () => {
  it.each([
    '../base-setting/BaseSettingMasterListField.vue',
    '../base-setting/BaseSettingNoticeTargetsField.vue',
    '../custom-text/CustomTextEntryCard.vue',
    '../js/JsConfigItemEditor.vue',
  ])('uses the shared repeatable structure in %s', componentPath => {
    const source = readComponent(componentPath);

    expect(source).toMatch(/<RepeatableList/);
    expect(source).toMatch(/<RepeatableItem/);
    expect(source).not.toMatch(/i-tabler-circle-plus-filled/);
    expect(source).not.toMatch(/i-tabler-circle-x(?!-filled)/);
  });

  it('keeps custom text cards compact and labels their feedback clearly', () => {
    const editor = readComponent('../custom-text/CustomTextEditor.vue');
    const card = readComponent('../custom-text/CustomTextEntryCard.vue');

    expect(editor).not.toMatch(/isMultiValue|:span=/);
    expect(card).toMatch(/add-label="添加文案"/);
    expect(card).toMatch(/i-tabler-circle-check-filled/);
    expect(card).toMatch(/color: var\(--sd-success\)/);
    expect(card).toMatch(/可用变量：/);
  });

  it('keeps the custom reply workspace and empty rule state consistent', () => {
    const workspace = readComponent('../custom-reply/CustomReplyEditor.vue');
    const rules = readComponent('../custom-reply/ReplyRulesSection.vue');

    expect(workspace).toMatch(/border-radius: var\(--sd-radius-md\)/);
    expect(workspace).toMatch(/overflow: hidden/);
    expect(rules).toMatch(/:empty="!rules\.length"/);
    expect(rules).toMatch(/empty-text="当前无规则"/);
  });

  it('keeps nested reply conditions removable and reports edits', () => {
    const editor = readComponent('../custom-reply/NestedRuleEditor.vue');
    const builder = readComponent('../custom-reply/ConditionBuilder.vue');

    expect(editor).toMatch(/@delete-condition="deleteAnyItem\(el\.conditions, \$event\)"/);
    expect(editor).toMatch(/<n-switch/);
    expect(editor).toMatch(/i-tabler-trash/);
    expect(editor).toMatch(/add-label="添加回复"/);
    expect(editor).not.toMatch(/添加随机回复/);
    expect(builder).toMatch(/watch\(listModel, \(\) => emit\('change'\)/);
  });
});

describe('list workspace contract', () => {
  it('provides one vertical rhythm for filters, result panels and pagination', () => {
    const source = readComponent('ListWorkspace.vue');

    expect(source).toMatch(/flex-direction: column/);
    expect(source).toMatch(/gap: var\(--sd-space-md\)/);
  });
});
