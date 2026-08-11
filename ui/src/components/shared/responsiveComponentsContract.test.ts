import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

describe('ResponsiveDataView contract', () => {
  it('measures its own container and renders one explicit view mode', () => {
    const source = readComponent('ResponsiveDataView.vue');

    expect(source).toMatch(/useResponsiveContainerMode/);
    expect(source).toMatch(/data-view-mode/);
    expect(source).toMatch(/v-if="mode === 'table'"/);
    expect(source).toMatch(/<slot[^>]+name="table"/);
    expect(source).toMatch(/<slot[^>]+name="compact"/);
  });
});

describe('ResponsiveTabs contract', () => {
  it('uses tabs for the full view and a select for the compact view', () => {
    const source = readComponent('ResponsiveTabs.vue');

    expect(source).toMatch(/<ResponsiveDataView/);
    expect(source).toMatch(/<n-tabs/);
    expect(source).toMatch(/<n-select/);
    expect(source).toMatch(/defineModel<string>\('value'/);
    expect(source).toMatch(/name="panel"/);
  });
});
