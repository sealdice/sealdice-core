export interface StoryPainterAdvancedModeSupportOptions {
  hasBlobArrayBuffer: boolean;
  hasFileReader: boolean;
  hasResizeObserver: boolean;
  hasTextEncoder: boolean;
  hasTextDecoder: boolean;
}

export interface StoryPainterAdvancedModeSupportResult {
  supported: boolean;
  reason?: string;
}

export function detectStoryPainterAdvancedModeSupport(
  options: StoryPainterAdvancedModeSupportOptions,
): StoryPainterAdvancedModeSupportResult {
  if (!options.hasResizeObserver) {
    return {
      supported: false,
      reason: '当前浏览器缺少 ResizeObserver，无法启用高级日志预览。',
    };
  }

  if (!options.hasBlobArrayBuffer && !options.hasFileReader) {
    return {
      supported: false,
      reason: '当前浏览器缺少 Blob/FileReader 读取能力，无法加载高级日志模式。',
    };
  }

  if (!options.hasTextEncoder || !options.hasTextDecoder) {
    return {
      supported: false,
      reason: '当前浏览器缺少文本编码能力，无法启用高级日志预览。',
    };
  }

  return { supported: true };
}

export function getStoryPainterAdvancedModeSupport(): StoryPainterAdvancedModeSupportResult {
  const BlobCtor = globalThis.Blob as (typeof Blob & { prototype?: Blob }) | undefined;
  return detectStoryPainterAdvancedModeSupport({
    hasBlobArrayBuffer: typeof BlobCtor?.prototype?.arrayBuffer === 'function',
    hasFileReader: typeof globalThis.FileReader !== 'undefined',
    hasResizeObserver: typeof globalThis.ResizeObserver !== 'undefined',
    hasTextEncoder: typeof globalThis.TextEncoder !== 'undefined',
    hasTextDecoder: typeof globalThis.TextDecoder !== 'undefined',
  });
}
