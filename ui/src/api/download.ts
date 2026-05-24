type DownloadResult = {
  data: unknown;
  headers: Record<string, unknown>;
};

function parseFilenameFromDisposition(disposition: string | null): string {
  if (!disposition) return '';

  const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (encoded?.[1]) {
    try {
      return decodeURIComponent(encoded[1]).trim();
    } catch {
      return encoded[1].trim();
    }
  }

  const plain = disposition.match(/filename="?([^";]+)"?/i);
  return plain?.[1]?.trim() ?? '';
}

function getHeader(headers: Record<string, unknown>, name: string): string | null {
  const target = name.toLowerCase();
  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() !== target) continue;
    if (typeof value === 'string') return value;
    if (Array.isArray(value)) return value.map(String).join(', ');
    if (value === undefined || value === null) return null;
    return String(value);
  }
  return null;
}

export function getDownloadFilename(
  response: { headers: Record<string, unknown> },
  suggestedFilename?: string,
): string {
  return (
    suggestedFilename?.trim() ||
    parseFilenameFromDisposition(getHeader(response.headers, 'content-disposition')) ||
    'download'
  );
}

function saveBlobFile(blob: Blob, filename: string): void {
  const objectUrl = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = objectUrl;
  link.download = filename;
  link.rel = 'noopener';
  link.click();
  URL.revokeObjectURL(objectUrl);
}

export async function downloadApiFile(
  resultPromise: Promise<DownloadResult>,
  suggestedFilename?: string,
): Promise<void> {
  const result = await resultPromise;
  if (!(result.data instanceof Blob)) {
    throw new Error('下载响应不是文件');
  }

  const filename = getDownloadFilename(result, suggestedFilename);

  saveBlobFile(result.data, filename);
}
