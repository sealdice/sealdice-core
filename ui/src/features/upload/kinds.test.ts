import { describe, expect, it } from 'vitest';
import { isFileAllowed, UPLOAD_KINDS } from './kinds';

describe('upload kinds', () => {
  it('registers every upload kind', () => {
    expect(Object.keys(UPLOAD_KINDS).sort()).toEqual([
      'censor',
      'custom-reply',
      'deck',
      'extension-install',
      'helpdoc',
      'js',
      'resource',
    ]);
  });

  it('restricts resource to image extensions', () => {
    expect(isFileAllowed('resource', 'a.png')).toBe(true);
    expect(isFileAllowed('resource', 'a.jpeg')).toBe(true);
    expect(isFileAllowed('resource', 'a.txt')).toBe(false);
  });

  it('restricts censor to txt/toml', () => {
    expect(isFileAllowed('censor', 'w.txt')).toBe(true);
    expect(isFileAllowed('censor', 'w.toml')).toBe(true);
    expect(isFileAllowed('censor', 'w.json')).toBe(false);
  });

  it('restricts helpdoc to json/xlsx', () => {
    expect(isFileAllowed('helpdoc', 'h.json')).toBe(true);
    expect(isFileAllowed('helpdoc', 'h.xlsx')).toBe(true);
    expect(isFileAllowed('helpdoc', 'h.txt')).toBe(false);
  });

  it('accepts any file when no extensions are declared', () => {
    expect(isFileAllowed('js', 'plugin.js')).toBe(true);
    expect(isFileAllowed('js', 'weird.xyz')).toBe(true);
  });

  it('helpdoc carries the group meta', () => {
    expect(UPLOAD_KINDS.helpdoc.extraMeta?.({ group: 'g1' })).toEqual({ group: 'g1' });
    expect(UPLOAD_KINDS.js.extraMeta).toBeUndefined();
  });

  it('does not queue the extension preview (it stays a direct huma call)', () => {
    expect(Object.keys(UPLOAD_KINDS)).not.toContain('extension-preview');
  });
});
