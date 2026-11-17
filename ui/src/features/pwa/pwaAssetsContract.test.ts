import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const projectRoot = resolve(fileURLToPath(new URL('../../..', import.meta.url)));

function readPngSize(path: string): { width: number; height: number } {
  const data = readFileSync(resolve(projectRoot, path));
  expect(data.subarray(1, 4).toString('ascii')).toBe('PNG');
  return {
    width: data.readUInt32BE(16),
    height: data.readUInt32BE(20),
  };
}

describe('PWA brand assets', () => {
  it('provides raster icons at the install sizes used by major platforms', () => {
    expect(readPngSize('public/pwa-192.png')).toEqual({ width: 192, height: 192 });
    expect(readPngSize('public/pwa-512.png')).toEqual({ width: 512, height: 512 });
    expect(readPngSize('public/pwa-maskable-512.png')).toEqual({ width: 512, height: 512 });
    expect(readPngSize('public/apple-touch-icon.png')).toEqual({ width: 180, height: 180 });
    expect(existsSync(resolve(projectRoot, 'public/favicon.ico'))).toBe(true);
  });
});
