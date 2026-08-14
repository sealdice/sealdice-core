import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

describe('editable collection contract', () => {
  it('keeps add actions below the items and uses the shared primary-secondary action', () => {
    const source = readComponent('RepeatableList.vue');

    expect(source.indexOf('sd-repeatable-list__items')).toBeLessThan(
      source.indexOf('sd-repeatable-list__footer')
    );
    expect(source).toMatch(/type="primary"/);
    expect(source).toMatch(/secondary/);
    expect(source).toMatch(/i-tabler-plus/);
  });

  it('keeps enable and destructive controls in the item header', () => {
    const source = readComponent('RepeatableItem.vue');

    expect(source).toMatch(/sd-repeatable-item__header/);
    expect(source).toMatch(/<n-switch/);
    expect(source).toMatch(/type="error"/);
    expect(source).toMatch(/i-tabler-trash/);
    expect(source).toMatch(/aria-label="removeLabel"/);
  });
});

describe('list workspace contract', () => {
  it('provides one vertical rhythm for filters, result panels and pagination', () => {
    const source = readComponent('ListWorkspace.vue');

    expect(source).toMatch(/flex-direction: column/);
    expect(source).toMatch(/gap: var\(--sd-space-md\)/);
  });
});
