import { readdirSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const sourceRoot = path.resolve(currentDir, '../..');

function collectVueFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) return collectVueFiles(entryPath);
    return entry.isFile() && entry.name.endsWith('.vue') ? [entryPath] : [];
  });
}

describe('password input contract', () => {
  it('uses persistent click toggles for every password or sensitive input', () => {
    const passwordInputs = collectVueFiles(sourceRoot).flatMap(file => {
      const source = readFileSync(file, 'utf8');
      return [...source.matchAll(/<n-input\b[\s\S]*?\/>/g)]
        .map(match => match[0])
        .filter(input => /password/.test(input));
    });

    expect(passwordInputs).toHaveLength(6);
    for (const input of passwordInputs) {
      expect(input).toContain('show-password-on="click"');
      expect(input).not.toContain('show-password-on="mousedown"');
    }
  });
});
