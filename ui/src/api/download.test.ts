import { getDownloadFilename } from './download';

function assertEqual(actual: unknown, expected: unknown): void {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
}

assertEqual(
  getDownloadFilename({
    headers: {
      'content-disposition': "attachment; filename*=UTF-8''%E6%B5%8B%E8%AF%95.json",
    },
  }, ''),
  '测试.json',
);

assertEqual(
  getDownloadFilename({
    headers: {
      'Content-Disposition': 'attachment; filename="backup.zip"',
    },
  }, ''),
  'backup.zip',
);

assertEqual(
  getDownloadFilename({
    headers: {},
  }, 'manual.txt'),
  'manual.txt',
);
