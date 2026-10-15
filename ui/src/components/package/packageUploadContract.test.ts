import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const source = readFileSync(path.resolve(currentDir, 'PackageManagerView.vue'), 'utf8');

describe('package upload wiring', () => {
  it('queues install through the global upload queue', () => {
    expect(source).toMatch(/enqueueUploads\(\[.*\],\s*'extension-install'/);
  });

  it('keeps the preview as a direct huma call', () => {
    expect(source).toMatch(/postSdApiV2ExtensionPreviewUpload/);
  });
});
