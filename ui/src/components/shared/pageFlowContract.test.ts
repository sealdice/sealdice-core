import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const sharedDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(sharedDir, '../..');

function readSource(relativePath: string): string {
  return readFileSync(path.resolve(srcDir, relativePath), 'utf8');
}

const ordinaryPages = [
  'pages/index.vue',
  'pages/connect.vue',
  'pages/misc/advanced-setting.vue',
  'pages/misc/backup.vue',
  'pages/misc/ban.vue',
  'pages/misc/base-setting.vue',
  'pages/misc/dice-public.vue',
  'pages/misc/group.vue',
  'pages/mod/censor.vue',
  'pages/mod/deck.vue',
  'pages/mod/helpdoc.vue',
  'pages/mod/js.vue',
  'pages/mod/story.vue',
  'pages/tool/profile.vue',
  'pages/tool/resource.vue',
  'pages/tool/test.vue',
  'components/custom-text/CustomTextEditor.vue',
  'components/package/PackageManagerView.vue',
];

const queryListSurfaces = [
  'pages/misc/group.vue',
  'pages/mod/deck.vue',
  'pages/mod/story.vue',
  'components/ban/BanListPanel.vue',
  'components/helpdoc/HelpdocItemPane.vue',
  'components/js/JsDataView.vue',
  'components/js/JsListView.vue',
  'components/resource/ResourceListPanel.vue',
];

describe('page vertical rhythm contract', () => {
  it('keeps external spacing out of PageHeader', () => {
    const source = readSource('components/shared/PageHeader.vue');

    expect(source).not.toMatch(/margin-bottom/);
  });

  it.each(ordinaryPages)('uses the shared page flow in %s', relativePath => {
    expect(readSource(relativePath)).toMatch(/sd-page-flow/);
  });

  it('does not add ad-hoc tab pane margins', () => {
    const jsPage = readSource('pages/mod/js.vue');
    const baseSetting = readSource('pages/misc/base-setting.vue');

    expect(jsPage).not.toMatch(/pane-class="[^"]*mb-/);
    expect(baseSetting).not.toMatch(/\.setting-tabs\s*\{[^}]*margin-top/s);
  });
});

describe('page list composition contract', () => {
  it.each(queryListSurfaces)('uses ListWorkspace around its list flow in %s', relativePath => {
    const source = readSource(relativePath);

    expect(source).toMatch(/<QueryToolbar/);
    expect(source).toMatch(/<ListWorkspace/);
  });

  it.each([
    'components/backup/BackupFileList.vue',
    'components/censor/CensorFilesView.vue',
    'components/censor/CensorLogView.vue',
    'components/censor/CensorWordsView.vue',
    'components/package/PackageManagerView.vue',
  ])('uses the shared rounded result panel in %s', relativePath => {
    expect(readSource(relativePath)).toMatch(/<ListPanel/);
  });
});
