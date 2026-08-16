import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

function readSource(relativePath: string): string {
  return readFileSync(path.resolve(currentDir, relativePath), 'utf8');
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
    expect(source).toMatch(/<n-tab/);
    expect(source).toMatch(/<n-select/);
    expect(source).toMatch(/defineModel<string>\('value'/);
    expect(source).toMatch(/name="panel"/);
    expect(source.match(/<slot[^>]+name="panel"/g)).toHaveLength(1);
  });

  it('uses the shared overflow rule for every remaining direct tab bar', () => {
    const componentPaths = [
      '../package/PackageManagerView.vue',
      '../../pages/misc/backup.vue',
      '../../pages/misc/ban.vue',
      '../../pages/mod/censor.vue',
      '../../pages/mod/helpdoc.vue',
      '../../pages/mod/story.vue',
      'ResponsiveTabs.vue',
    ];

    for (const componentPath of componentPaths) {
      const source = readSource(componentPath);
      const tabBars = [...source.matchAll(/<n-tabs\b[\s\S]*?>/g)].map(match => match[0]);
      expect(tabBars.length, componentPath).toBeGreaterThan(0);
      for (const tabBar of tabBars) {
        expect(tabBar, componentPath).toContain('sd-scrollable-tabs');
        expect(tabBar, componentPath).not.toContain('space-evenly');
      }
    }

    const baseCss = readSource('../../assets/base.css');
    expect(baseCss).toMatch(/\.sd-scrollable-tabs \.n-tabs-nav-scroll-content/);
    expect(baseCss).toMatch(/min-width:\s*max-content/);
    expect(baseCss).toMatch(/\.sd-scrollable-tabs \.n-tabs-tab[\s\S]*?flex:\s*none/);
  });

  it('uses compact selectors for the four JS workspaces and dynamic config groups', () => {
    const jsPage = readSource('../../pages/mod/js.vue');
    const jsConfigGroups = readSource('../js/JsConfigGroups.vue');
    const jsTabOptions = jsPage.match(/const jsTabOptions = \[[\s\S]*?\];/)?.[0] ?? '';

    expect(jsPage).toMatch(/<ResponsiveTabs/);
    expect(jsPage).not.toMatch(/<n-tabs/);
    expect(jsTabOptions.match(/label:\s*'[^']+'/g)).toHaveLength(4);
    expect(jsConfigGroups).toMatch(/<ResponsiveTabs/);
    expect(jsConfigGroups).toMatch(/:options="groupOptions"/);
  });
});
