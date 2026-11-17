export type RuntimeSummaryInput = {
  OS?: string;
  arch?: string;
  containerMode?: boolean;
  justForTest?: boolean;
};

/**
 * 运行环境各段之间使用全角中点：中文语境下的分隔符，自带左右空隙，两侧不再补空格。
 * 这里的 OS 与架构是取值而非数据格式分隔符，因此适用排版规范。
 */
export const RUNTIME_SEGMENT_SEPARATOR = '・';

const RUNTIME_FALLBACK_TEXT = '读取中';

/** 运行模式的完整口径，用于「运行模式」这类独立字段。 */
export function formatRuntimeMode(runtime: RuntimeSummaryInput | undefined): string {
  if (!runtime) return RUNTIME_FALLBACK_TEXT;
  if (runtime.justForTest) return '展示模式';
  if (runtime.containerMode) return '容器模式';
  return '本机运行';
}

/**
 * 运行环境摘要：`OS・架构`，`withMode` 时在非本机运行的情况下追加运行模式。
 * 本机运行是默认状态，不占用一段文字。
 */
export function formatRuntimeSummary(
  runtime: RuntimeSummaryInput | undefined,
  options: { withMode?: boolean } = {}
): string {
  if (!runtime?.OS || !runtime.arch) return RUNTIME_FALLBACK_TEXT;

  const segments = [runtime.OS, runtime.arch];
  if (options.withMode && (runtime.justForTest || runtime.containerMode)) {
    segments.push(formatRuntimeMode(runtime));
  }
  return segments.join(RUNTIME_SEGMENT_SEPARATOR);
}
