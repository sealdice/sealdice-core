import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const responsiveDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(responsiveDir, '../..');

const cases = [
  ['pages/tool/resource.vue', 420],
  ['components/package/PackageDetailDrawer.vue', 720],
  ['pages/tool/profile.vue', 880],
] as const;

describe('responsive overlay width contract', () => {
  for (const [relativePath, maxWidth] of cases) {
    it(`${relativePath} keeps ${maxWidth}px as a maximum with safe viewport gutters`, () => {
      const source = readFileSync(path.resolve(srcDir, relativePath), 'utf8');

      expect(source).toMatch(/useResponsiveOverlayWidth/);
      expect(source).toMatch(new RegExp(`maxWidth:\\s*${maxWidth}`));
      expect(source).toMatch(/gutter:\s*16/);
      expect(source).toMatch(/:width=/);
    });
  }
});
