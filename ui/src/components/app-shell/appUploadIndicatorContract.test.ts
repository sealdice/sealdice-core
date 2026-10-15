import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

describe('upload indicator contract', () => {
  it('renders a badge with an animated upload icon and queue actions', () => {
    const source = readFileSync(path.resolve(currentDir, 'AppUploadIndicator.vue'), 'utf8');
    expect(source).toMatch(/n-badge/);
    expect(source).toMatch(/i-tabler-cloud-upload/);
    expect(source).toMatch(/sd-upload-indicator__spin|animation/);
    expect(source).toMatch(/全部取消/);
    expect(source).toMatch(/取消/);
    expect(source).toMatch(/重试/);
  });

  it('is mounted in the top breadcrumb bar', () => {
    const source = readFileSync(path.resolve(currentDir, 'AppBreadcrumb.vue'), 'utf8');
    expect(source).toMatch(/AppUploadIndicator/);
  });
});
