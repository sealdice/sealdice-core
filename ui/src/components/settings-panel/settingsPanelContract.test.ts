import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

function readComponent(name: string): string {
  return readFileSync(path.resolve(currentDir, name), 'utf8');
}

describe('settings panel framework contract', () => {
  it('routes every settings category through the shared field layout', () => {
    const source = readComponent('SettingCategoryBox.vue');

    expect(source).toMatch(/<SettingFieldLayout/);
    expect(source).toMatch(/:columns="columns"/);
    expect(source).toMatch(/:padded="padded"/);
  });

  it('keeps intermediate feedback space and collapses only an empty final row', () => {
    const source = readComponent('SettingFieldLayout.vue');

    expect(source).toMatch(/n-form-item:last-child/);
    expect(source).toMatch(/n-form-item:nth-last-child\(2\):nth-child\(odd\)/);
    expect(source).toMatch(/feedback-wrapper:empty/);
    expect(source).not.toMatch(/show-feedback="false"/);
  });
});
