type DownloadResult = {
  data: unknown;
  request: Request;
  response: Response;
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

  const filename =
    suggestedFilename?.trim() ||
    parseFilenameFromDisposition(result.response.headers.get('content-disposition')) ||
    'download';

  saveBlobFile(result.data, filename);
}
