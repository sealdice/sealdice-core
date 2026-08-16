import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

describe('custom text presentation contract', () => {
  it('balances variable-height entry cards into responsive columns', () => {
    const source = readComponent('CustomTextEditor.vue');

    expect(source).toMatch(/class="entry-columns"/);
    expect(source).toMatch(/class="entry-column-item"/);
    expect(source).toMatch(/column-count: 1/);
    expect(source).toMatch(/@media \(min-width: 1024px\)[\s\S]*column-count: 2/);
    expect(source).toMatch(/break-inside: avoid/);
    expect(source).not.toMatch(/cols="1 m:2"/);
  });

  it('renders the multi-value disclosure as a clear secondary action', () => {
    const source = readComponent('CustomTextEntryCard.vue');

    expect(source).toMatch(/#footer-extra[\s\S]*v-if="isLongList"[\s\S]*secondary/);
    expect(source).toMatch(/i-tabler-chevron-down/);
    expect(source).toMatch(/i-tabler-chevron-up/);
    expect(source).toMatch(/展开更多（其余/);
    expect(source).toMatch(/:aria-expanded="!collapsed"/);
    expect(source).toMatch(/function handleAddItem\(\)[\s\S]*collapsed\.value = false/);
  });
});
