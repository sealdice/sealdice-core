import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

it('derives the shell mode from live viewport width and syncs sidebar defaults on mode changes', () => {
  const source = readFileSync(path.resolve(currentDir, 'AppShell.vue'), 'utf8');

  expect(source).toMatch(/useWindowSize/);
  expect(source).toMatch(/getAppShellViewportMode/);
  expect(source).toMatch(/watch\(viewportMode/);
  expect(source).toMatch(/:viewport-mode="viewportMode"/);
  expect(source).toMatch(/viewportMode\.value === 'mobile'/);
});
