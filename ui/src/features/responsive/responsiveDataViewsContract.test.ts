import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const responsiveDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(responsiveDir, '../..');

function readSource(relativePath: string): string {
  return readFileSync(path.resolve(srcDir, relativePath), 'utf8');
}

describe('responsive data view coverage', () => {
  const cases = [
    ['components/package/PackageInstalledDataView.vue', 760],
    ['components/package/PackageStoreDataView.vue', 760],
    ['components/backup/BackupFileList.vue', 680],
    ['components/censor/CensorFilesView.vue', 560],
    ['components/censor/CensorWordsView.vue', 520],
    ['components/censor/CensorLogView.vue', 960],
    ['components/helpdoc/HelpdocItemPane.vue', 900],
    ['components/public-dice/PublicDiceEndpointSelector.vue', 760],
  ] as const;

  for (const [relativePath, compactAt] of cases) {
    it(`${relativePath} switches from the data table at ${compactAt}px`, () => {
      const source = readSource(relativePath);

      expect(source).toMatch(/<ResponsiveDataView/);
      expect(source).toMatch(new RegExp(`:compact-at="${compactAt}"`));
      expect(source).toMatch(/#table/);
      expect(source).toMatch(/#compact/);
    });
  }

  it('keeps package orchestration in the parent and delegates both data presentations', () => {
    const source = readSource('components/package/PackageManagerView.vue');

    expect(source).toMatch(/<PackageInstalledDataView/);
    expect(source).toMatch(/<PackageStoreDataView/);
    expect(source).not.toMatch(/const installedColumns/);
    expect(source).not.toMatch(/const storeColumns/);
  });
});
