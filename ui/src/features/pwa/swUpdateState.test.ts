import { describe, expect, it } from 'vitest';
import {
  CONTROLLED_RELOAD_WINDOW_MS,
  isChunkLoadError,
  parseReloadGuardTimestamp,
  shouldAllowControlledReload,
} from './swUpdateState';

describe('isChunkLoadError', () => {
  it('识别 Chrome 的动态 import 失败', () => {
    const error = new TypeError(
      'Failed to fetch dynamically imported module: http://x/assets/about-A.js'
    );
    expect(isChunkLoadError(error)).toBe(true);
  });

  it('识别 Firefox 的动态 import 失败', () => {
    const error = new TypeError(
      'error loading dynamically imported module: http://x/assets/about-A.js'
    );
    expect(isChunkLoadError(error)).toBe(true);
  });

  it('识别 Safari 的模块脚本失败', () => {
    expect(isChunkLoadError(new Error('Importing a module script failed.'))).toBe(true);
  });

  it('识别 ChunkLoadError 名称', () => {
    const error = new Error('whatever');
    error.name = 'ChunkLoadError';
    expect(isChunkLoadError(error)).toBe(true);
  });

  it('普通业务错误不算 chunk 加载失败', () => {
    expect(isChunkLoadError(new Error('Network Error'))).toBe(false);
    expect(isChunkLoadError(new Error('navigation cancelled'))).toBe(false);
  });

  it('非 Error 输入不算 chunk 加载失败', () => {
    expect(isChunkLoadError(undefined)).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError('Failed to fetch dynamically imported module')).toBe(false);
    expect(isChunkLoadError(42)).toBe(false);
  });
});

describe('shouldAllowControlledReload', () => {
  it('从未刷新过时允许', () => {
    expect(shouldAllowControlledReload(null, 1000)).toBe(true);
  });

  it('窗口期内禁止重复刷新', () => {
    const now = 100_000;
    expect(shouldAllowControlledReload(now - CONTROLLED_RELOAD_WINDOW_MS + 1, now)).toBe(false);
  });

  it('超过窗口期后允许再次刷新', () => {
    const now = 100_000;
    expect(shouldAllowControlledReload(now - CONTROLLED_RELOAD_WINDOW_MS, now)).toBe(true);
  });

  it('非法时间戳按从未刷新处理', () => {
    expect(shouldAllowControlledReload(Number.NaN, 1000)).toBe(true);
  });
});

describe('parseReloadGuardTimestamp', () => {
  it('解析合法时间戳', () => {
    expect(parseReloadGuardTimestamp('1720000000000')).toBe(1720000000000);
  });

  it('空值与非法值返回 null', () => {
    expect(parseReloadGuardTimestamp(null)).toBe(null);
    expect(parseReloadGuardTimestamp('')).toBe(null);
    expect(parseReloadGuardTimestamp('abc')).toBe(null);
    expect(parseReloadGuardTimestamp('0')).toBe(null);
    expect(parseReloadGuardTimestamp('-5')).toBe(null);
  });
});
