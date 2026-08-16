export function getTestModeWatermarkRows(content: string, rows = 7): string[] {
  const text = content.trim();
  if (!text) return [];
  return Array.from({ length: rows }, (_, index) => `${text} · ${text} · ${text} · ${index + 1}`);
}
