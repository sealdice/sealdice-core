import { blobToArrayBuffer, hashFile, sha256Hex } from './fileHash';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

const nativeDigest = await sha256Hex(new TextEncoder().encode('hello compatibility').buffer);
assertEqual(nativeDigest, '64747d86a3ddfc265f70df068d330fcaa13b7c593ba7b49cf99692bdc7053ac6');

const originalCrypto = globalThis.crypto;
Object.defineProperty(globalThis, 'crypto', {
  configurable: true,
  value: undefined,
});

const fallbackDigest = await sha256Hex(new TextEncoder().encode('hello compatibility').buffer);
assertEqual(fallbackDigest, nativeDigest);

const originalFileReader = globalThis.FileReader;
const expectedBytes = Uint8Array.from([1, 2, 3, 4]);

class FakeFileReader {
  result: ArrayBuffer | null = null;
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;

  readAsArrayBuffer(_blob: Blob) {
    this.result = expectedBytes.buffer.slice(0);
    this.onload?.();
  }
}

Object.defineProperty(globalThis, 'FileReader', {
  configurable: true,
  value: FakeFileReader,
});

const fallbackBytes = await blobToArrayBuffer({} as Blob);
assertEqual(JSON.stringify(Array.from(new Uint8Array(fallbackBytes))), JSON.stringify(Array.from(expectedBytes)));

Object.defineProperty(globalThis, 'crypto', {
  configurable: true,
  value: originalCrypto,
});

Object.defineProperty(globalThis, 'FileReader', {
  configurable: true,
  value: originalFileReader,
});

const fileHash = await hashFile(new Blob(['story upload']));
assertEqual(fileHash, '57a8501edefe84ba3c84a188bef478874f10d25243e1be50b7bd9dc8e5892433');
