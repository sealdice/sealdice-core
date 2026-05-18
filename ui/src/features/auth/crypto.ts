const passwordHashVersion = 'v01';
const defaultIterations = 1000;
const derivedBitsLength = 256;

function bytesToBinaryString(bytes: number[]): string {
  return bytes.map(byte => String.fromCharCode(byte)).join('');
}

function iterationsToBytes(iterations: number): number[] {
  const hex = iterations.toString(16).padStart(6, '0').slice(-6);
  const bytes = hex.match(/.{2}/g) ?? [];
  return bytes.map(byte => Number.parseInt(byte, 16));
}

function encodeBinaryText(text: string): string {
  return btoa(text);
}

function serializePasswordHash(keyBuffer: ArrayBuffer, saltBytes: Uint8Array, iterations: number): string {
  const payload = [
    ...Array.from(saltBytes),
    ...iterationsToBytes(iterations),
    ...Array.from(new Uint8Array(keyBuffer)),
  ];
  return encodeBinaryText(`${passwordHashVersion}${bytesToBinaryString(payload)}`);
}

export async function passwordHash(
  salt: string,
  password: string,
  iterations = defaultIterations,
): Promise<string> {
  const cryptoApi = globalThis.crypto?.subtle;
  if (!cryptoApi) {
    throw new Error('当前环境不支持 Web Crypto，无法生成登录密码哈希。');
  }

  const encoder = new TextEncoder();
  const passwordKey = await cryptoApi.importKey(
    'raw',
    encoder.encode(password),
    'PBKDF2',
    false,
    ['deriveBits'],
  );
  const saltBytes = encoder.encode(salt);
  const keyBuffer = await cryptoApi.deriveBits(
    {
      name: 'PBKDF2',
      hash: 'SHA-512',
      salt: saltBytes,
      iterations,
    },
    passwordKey,
    derivedBitsLength,
  );

  return serializePasswordHash(keyBuffer, saltBytes, iterations);
}
