import { describe, expect, it } from 'vitest';
import { buildPwaInstallGuide, detectPwaBrowser, detectPwaPlatform } from './pwaInstallGuide';

describe('PWA install environment', () => {
  it('detects the supported platform families', () => {
    expect(
      detectPwaPlatform('Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/140.0 Safari/537.36')
    ).toBe('windows');
    expect(detectPwaPlatform('Mozilla/5.0 (Linux; Android 16) Chrome/140.0')).toBe('android');
    expect(detectPwaPlatform('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)')).toBe('ios');
    expect(detectPwaPlatform('Mozilla/5.0 (Macintosh; Intel Mac OS X)', 'MacIntel', 5)).toBe('ios');
  });

  it('detects browsers without treating iOS variants as Safari', () => {
    expect(detectPwaBrowser('Mozilla/5.0 Firefox/141.0')).toBe('firefox');
    expect(detectPwaBrowser('Mozilla/5.0 FxiOS/141.0 Mobile Safari/605.1.15')).toBe('firefox');
    expect(detectPwaBrowser('Mozilla/5.0 CriOS/140.0 Mobile Safari/604.1')).toBe('chromium');
    expect(detectPwaBrowser('Mozilla/5.0 Version/18.0 Safari/605.1.15')).toBe('safari');
  });
});

describe('PWA install guidance', () => {
  it('explains why the Vite development server cannot trigger Chrome installation', () => {
    const guide = buildPwaInstallGuide({
      platform: 'windows',
      browser: 'chromium',
      isSecureContext: true,
      isDevelopment: true,
    });
    expect(guide.title).toContain('开发环境');
  });

  it('prioritizes the secure-context requirement', () => {
    const guide = buildPwaInstallGuide({
      platform: 'android',
      browser: 'chromium',
      isSecureContext: false,
      isDevelopment: false,
    });
    expect(guide.title).toContain('安装条件');
    expect(guide.warning).toContain('快捷方式');
  });

  it('provides platform-specific manual installation paths', () => {
    expect(
      buildPwaInstallGuide({
        platform: 'ios',
        browser: 'safari',
        isSecureContext: true,
        isDevelopment: false,
      }).steps
    ).toContain('选择“添加到主屏幕”。');
    expect(
      buildPwaInstallGuide({
        platform: 'macos',
        browser: 'safari',
        isSecureContext: true,
        isDevelopment: false,
      }).steps
    ).toContain('选择“添加到程序坞”。');
    expect(
      buildPwaInstallGuide({
        platform: 'linux',
        browser: 'firefox',
        isSecureContext: true,
        isDevelopment: false,
      }).title
    ).toContain('暂不支持');
  });
});
