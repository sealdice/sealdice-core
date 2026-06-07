import { detectStoryPainterAdvancedModeSupport } from './compat';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const supported = detectStoryPainterAdvancedModeSupport({
  hasBlobArrayBuffer: true,
  hasFileReader: false,
  hasResizeObserver: true,
  hasTextEncoder: true,
  hasTextDecoder: true,
});
assertEqual(supported.supported, true);

const missingBlobReader = detectStoryPainterAdvancedModeSupport({
  hasBlobArrayBuffer: false,
  hasFileReader: false,
  hasResizeObserver: true,
  hasTextEncoder: true,
  hasTextDecoder: true,
});
assertEqual(missingBlobReader.supported, false);
assertEqual(missingBlobReader.reason?.includes('Blob'), true);

const missingResizeObserver = detectStoryPainterAdvancedModeSupport({
  hasBlobArrayBuffer: true,
  hasFileReader: true,
  hasResizeObserver: false,
  hasTextEncoder: true,
  hasTextDecoder: true,
});
assertEqual(missingResizeObserver.supported, false);
assertEqual(missingResizeObserver.reason?.includes('ResizeObserver'), true);
