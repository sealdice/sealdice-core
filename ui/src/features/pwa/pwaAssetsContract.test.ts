import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const projectRoot = resolve(fileURLToPath(new URL('../../..', import.meta.url)));

function readProjectFile(path: string): string {
  return readFileSync(resolve(projectRoot, path), 'utf8');
}

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

  it('references normal, maskable, scalable, and Apple icons', () => {
    const viteConfig = readProjectFile('vite.config.ts');
    const html = readProjectFile('index.html');

    expect(viteConfig).toContain("lang: 'zh-CN'");
    expect(viteConfig).toContain("src: './pwa-192.png'");
    expect(viteConfig).toContain("src: './pwa-512.png'");
    expect(viteConfig).toContain("src: './pwa-maskable-512.png'");
    expect(viteConfig).toContain("src: './pwa-icon.svg'");
    expect(viteConfig).toMatch(/pwa-maskable-512\.png'[\s\S]*purpose: 'maskable'/);
    expect(html).toContain('rel="apple-touch-icon"');
  });

  it('keeps installation discoverable when no native prompt is available', () => {
    const installButton = readProjectFile('src/components/app-shell/AppInstallButton.vue');

    expect(installButton).toContain("return '安装为应用'");
    expect(installButton).toContain('guideVisible.value = true');
    expect(installButton).not.toMatch(/<n-tooltip\s+v-if=/);
  });
});
