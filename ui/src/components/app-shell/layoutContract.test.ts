import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(currentDir, '../..');
const assetsDir = path.resolve(srcDir, 'assets');

it('uses shared content-width and spacing tokens for the app shell layout contract', async () => {
  const appShellSource = readFileSync(path.resolve(currentDir, 'AppShell.vue'), 'utf8');
  const baseCssSource = readFileSync(path.resolve(assetsDir, 'base.css'), 'utf8');
  const mainCssSource = readFileSync(path.resolve(assetsDir, 'main.css'), 'utf8');

  const assertMatch = (source: string, pattern: RegExp, label: string) => {
    if (!pattern.test(source)) {
      throw new Error(`expected ${label}`);
    }
  };

  const assertNotMatch = (source: string, pattern: RegExp, label: string) => {
    if (pattern.test(source)) {
      throw new Error(`expected not ${label}`);
    }
  };

  assertMatch(baseCssSource, /--sd-content-max-width:\s*1180px;/, 'base.css to define --sd-content-max-width');
  assertMatch(baseCssSource, /--sd-space-md:\s*1rem;/, 'base.css to define --sd-space-md');
  assertMatch(baseCssSource, /--sd-space-xl:\s*1\.5rem;/, 'base.css to define --sd-space-xl');
  assertNotMatch(mainCssSource, /text-align:\s*center;/, 'main.css to keep #app text-align center');
  assertMatch(
    appShellSource,
    /\.sd-main-container\s*\{[\s\S]*max-width:\s*var\(--sd-content-max-width\);/m,
    'AppShell.vue to apply shared content max width in the default shell container',
  );
  assertMatch(
    appShellSource,
    /\.sd-main-container--wide\s*\{[\s\S]*max-width:\s*none;/m,
    'AppShell.vue to preserve wide pages as unconstrained containers',
  );
});
