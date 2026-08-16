import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const responsiveDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(responsiveDir, '../..');

function readSource(relativePath: string): string {
  return readFileSync(path.resolve(srcDir, relativePath), 'utf8');
}

describe('reported responsive issue contracts', () => {
  it('stacks notice target labels from their own 420px container', () => {
    const source = readSource('components/base-setting/BaseSettingNoticeTargetsField.vue');

    expect(source).toMatch(/container-type:\s*inline-size/);
    expect(source).toMatch(/@container\s*\(max-width:\s*420px\)/);
    expect(source.match(/class="notice-target-field"/g)).toHaveLength(2);
    expect(source).toMatch(/>ID<\/n-text>/);
    expect(source).toMatch(/>通知项<\/n-text>/);
    expect(source).toMatch(
      /\.notice-target-fields\s*\{[^}]*grid-template-columns: minmax\(0, 1fr\)/s
    );
    expect(source).not.toMatch(/@media screen and \(max-width: 767\.9px\)/);
  });

  it('makes all base-setting categories discoverable through responsive tabs', () => {
    const source = readSource('pages/misc/base-setting.vue');

    expect(source).toMatch(/<ResponsiveTabs/);
    expect(source).toMatch(/:compact-at="760"/);
    expect(source).toMatch(/:options="tabOptions"/);
  });

  it('keeps all wizard step names visible in the mobile progress summary', () => {
    const source = readSource('components/connect/ConnectCreateWizard.vue');

    expect(source).toMatch(/wizard-progress/);
    expect(source).toMatch(/remainingWizardStepTitles/);
    expect(source).not.toMatch(/\.n-step\s*>\s*\.n-step-content/);
  });

  it('targets Naive UI modal cards whether the class is on the card or an ancestor', () => {
    const source = readSource('assets/base.css');

    expect(source).toMatch(/\.the-dialog\.n-card/);
    expect(source).toMatch(/\.feed-modal\.n-card/);
  });

  it('stacks the reply editor from its own 720px container', () => {
    const editor = readSource('components/custom-reply/CustomReplyEditor.vue');
    const sidebar = readSource('components/custom-reply/ReplyFileSidebar.vue');

    expect(editor).toMatch(/container-name:\s*reply-editor/);
    expect(editor).toMatch(/\.reply-editor-container\s*\{[^}]*flex:\s*1 1 auto/s);
    expect(editor).toMatch(/\.reply-editor-container\s*\{[^}]*min-width:\s*0/s);
    expect(editor).toMatch(/@container reply-editor \(max-width:\s*720px\)/);
    expect(sidebar).toMatch(/@container reply-editor \(max-width:\s*720px\)/);
  });

  it('uses a two-by-two summary grid below the desktop breakpoint', () => {
    const source = readSource('pages/misc/group.vue');

    expect(source).toMatch(/cols="2 m:4"/);
    expect(source).not.toMatch(/cols="1 s:2 m:4"/);
  });
});
