type FileReaderCtor = typeof FileReader;

export async function blobToArrayBuffer(blob: Blob): Promise<ArrayBuffer> {
  if (typeof blob.arrayBuffer === 'function') {
    return await blob.arrayBuffer();
  }

  const FileReaderImpl = globalThis.FileReader as FileReaderCtor | undefined;
  if (!FileReaderImpl) {
    throw new Error('当前环境不支持读取文件内容');
  }

  return await new Promise<ArrayBuffer>((resolve, reject) => {
    const reader = new FileReaderImpl();
    reader.onerror = () => reject(reader.error ?? new Error('读取文件失败'));
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.readAsArrayBuffer(blob);
  });
}

export async function sha256Hex(buffer: ArrayBuffer): Promise<string> {
  const cryptoApi = globalThis.crypto?.subtle;
  if (cryptoApi) {
    const hash = await cryptoApi.digest('SHA-256', buffer);
    return bytesToHex(new Uint8Array(hash));
  }

  const { sha256 } = await import('js-sha256');
  return sha256(new Uint8Array(buffer));
}

export async function hashFile(file: Blob): Promise<string> {
  const buffer = await blobToArrayBuffer(file);
  return await sha256Hex(buffer);
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map(byte => byte.toString(16).padStart(2, '0'))
    .join('');
}
