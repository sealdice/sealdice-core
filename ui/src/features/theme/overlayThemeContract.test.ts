import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { createThemeOverrides } from './themePalette';

const themeDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.resolve(themeDir, '../..');

function collectSourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const absolutePath = path.join(directory, entry.name);
    if (entry.isDirectory()) return collectSourceFiles(absolutePath);
    return /\.(?:ts|vue)$/.test(entry.name) && !entry.name.endsWith('.test.ts')
      ? [absolutePath]
      : [];
  });
}

describe('overlay theme contract', () => {
  it('keeps template modals under the global themed providers', () => {
    const appSource = readFileSync(path.join(srcDir, 'App.vue'), 'utf8');
    const configProviderSources = collectSourceFiles(srcDir).filter(sourcePath =>
      /<ProConfigProvider|<n-config-provider|<NConfigProvider/.test(
        readFileSync(sourcePath, 'utf8')
      )
    );

    expect(configProviderSources).toEqual([path.join(srcDir, 'App.vue')]);
    expect(appSource).toMatch(/:theme="activeTheme"/);
    expect(appSource).toMatch(/:theme-overrides="themeOverrides"/);
    expect(appSource).toMatch(/<n-modal-provider>[\s\S]*<n-dialog-provider>/);
  });

  it('passes the live application theme to every discrete overlay API', () => {
    const discreteSources = collectSourceFiles(srcDir)
      .map(sourcePath => ({ sourcePath, source: readFileSync(sourcePath, 'utf8') }))
      .filter(({ source }) => source.includes('createDiscreteApi('));

    expect(
      discreteSources.map(({ sourcePath }) =>
        path.relative(srcDir, sourcePath).split(path.sep).join('/')
      )
    ).toEqual(['api/client.ts']);
    expect(discreteSources[0]?.source).toMatch(/configProviderProps: discreteConfigProviderProps/);
    expect(discreteSources[0]?.source).toMatch(/themeStore\.resolvedTheme === 'dark'/);
    expect(discreteSources[0]?.source).toMatch(/themeOverrides: themeStore\.themeOverrides/);
  });

  it('provides dark surfaces for cards, modals and popovers', () => {
    const darkOverrides = createThemeOverrides('dark');

    expect(darkOverrides.common?.cardColor).toBe('#182133');
    expect(darkOverrides.common?.modalColor).toBe('#182133');
    expect(darkOverrides.common?.popoverColor).toBe('#182133');
  });

  it('does not hard-code a light surface in template modal components', () => {
    const modalSources = collectSourceFiles(srcDir)
      .map(sourcePath => readFileSync(sourcePath, 'utf8'))
      .filter(source => /<n-modal|<NModal/.test(source));

    expect(modalSources.length).toBeGreaterThan(0);
    for (const source of modalSources) {
      expect(source).not.toMatch(
        /background(?:-color)?:\s*(?:#fff(?:fff)?\b|white\b|rgb\(255\s*,\s*255\s*,\s*255\))/i
      );
    }
  });
});
