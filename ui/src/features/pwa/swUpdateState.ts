// Service Worker 换版与异步 chunk 加载失败的纯逻辑判断。
// 副作用（注册 SW、刷新页面）在 swUpdate.ts，这里只保留可测试的判定规则。

export type SwUpdateReason = 'sw-update' | 'chunk-error';

export const CONTROLLED_RELOAD_AT_KEY = 'sd:sw-update-reload-at';
export const CONTROLLED_RELOAD_WINDOW_MS = 30_000;

const CHUNK_LOAD_ERROR_PATTERNS = [
  // Chrome / Edge
  /failed to fetch dynamically imported module/i,
  // Firefox
  /error loading dynamically imported module/i,
  // Safari
  /importing a module script failed/i,
];

// 路由页面是异步 chunk，版本切换后旧 hash 文件会 404。
// 各浏览器的动态 import 失败文案不同，这里统一识别。
export function isChunkLoadError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false;

  const { name, message } = error as { name?: unknown; message?: unknown };
  if (name === 'ChunkLoadError') return true;
  if (typeof message !== 'string') return false;

  return CHUNK_LOAD_ERROR_PATTERNS.some(pattern => pattern.test(message));
}

// 受控刷新的防循环窗口：上次刷新后短时间内再次触发时，改为提示用户手动处理。
export function shouldAllowControlledReload(
  lastReloadAt: number | null,
  now: number,
  windowMs: number = CONTROLLED_RELOAD_WINDOW_MS
): boolean {
  if (lastReloadAt === null || !Number.isFinite(lastReloadAt)) return true;
  return now - lastReloadAt >= windowMs;
}

export function parseReloadGuardTimestamp(raw: string | null): number | null {
  if (!raw) return null;
  const value = Number(raw);
  return Number.isFinite(value) && value > 0 ? value : null;
}
