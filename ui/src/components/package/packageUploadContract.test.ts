import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const source = readFileSync(path.resolve(currentDir, 'PackageManagerView.vue'), 'utf8');

describe('package upload wiring', () => {
  it('no longer calls the removed multipart install endpoint', () => {
    expect(source).not.toMatch(/postSdApiV2ExtensionInstallUpload/);
  });

  it('queues installs through the global upload queue', () => {
    expect(source).toMatch(/enqueueUploads\(/);
    expect(source).toMatch(/'extension-install'/);
  });
});
