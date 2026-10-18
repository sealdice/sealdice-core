import { getDownloadFilename } from './download';
import { expect, it } from 'vitest';

it('解析文件名并规范下载请求', async () => {
  expect(
    getDownloadFilename(
      {
        headers: {
          'content-disposition': "attachment; filename*=UTF-8''%E6%B5%8B%E8%AF%95.json",
        },
      },
      ''
    )
  ).toBe('测试.json');

  expect(
    getDownloadFilename(
      {
        headers: {
          'Content-Disposition': 'attachment; filename="backup.zip"',
        },
      },
      ''
    )
  ).toBe('backup.zip');

  expect(
    getDownloadFilename(
      {
        headers: {},
      },
      'manual.txt'
    )
  ).toBe('manual.txt');
});
