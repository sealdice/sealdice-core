import { detectStoryPainterAdvancedModeSupport } from './compat';
import { expect, it } from 'vitest';

it('日志阅读器兼容不同记录格式', async () => {
  const supported = detectStoryPainterAdvancedModeSupport({
    hasBlobArrayBuffer: true,
    hasFileReader: false,
    hasResizeObserver: true,
    hasTextEncoder: true,
    hasTextDecoder: true,
  });
  expect(supported.supported).toBe(true);

  const missingBlobReader = detectStoryPainterAdvancedModeSupport({
    hasBlobArrayBuffer: false,
    hasFileReader: false,
    hasResizeObserver: true,
    hasTextEncoder: true,
    hasTextDecoder: true,
  });
  expect(missingBlobReader.supported).toBe(false);
  expect(missingBlobReader.reason?.includes('Blob')).toBe(true);

  const missingResizeObserver = detectStoryPainterAdvancedModeSupport({
    hasBlobArrayBuffer: true,
    hasFileReader: true,
    hasResizeObserver: false,
    hasTextEncoder: true,
    hasTextDecoder: true,
  });
  expect(missingResizeObserver.supported).toBe(false);
  expect(missingResizeObserver.reason?.includes('ResizeObserver')).toBe(true);
});
