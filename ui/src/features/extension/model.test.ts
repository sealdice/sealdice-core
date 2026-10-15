import {
  buildExtensionAssetUrl,
  buildPackageFileTree,
  summarizeStoreInstallResult,
} from './model.js';
import { expect, it } from 'vitest';

it('扩展列表模型和配置转换', async () => {
  expect(
    buildPackageFileTree(['reply/main.yaml', 'reply/assets/icon.png', 'scripts/index.js'])
  ).toEqual([
    {
      key: 'reply',
      name: 'reply',
      path: 'reply',
      kind: 'directory',
      children: [
        {
          key: 'reply/assets',
          name: 'assets',
          path: 'reply/assets',
          kind: 'directory',
          children: [
            {
              key: 'reply/assets/icon.png',
              name: 'icon.png',
              path: 'reply/assets/icon.png',
              kind: 'file',
            },
          ],
        },
        { key: 'reply/main.yaml', name: 'main.yaml', path: 'reply/main.yaml', kind: 'file' },
      ],
    },
    {
      key: 'scripts',
      name: 'scripts',
      path: 'scripts',
      kind: 'directory',
      children: [
        { key: 'scripts/index.js', name: 'index.js', path: 'scripts/index.js', kind: 'file' },
      ],
    },
  ]);

  expect(
    summarizeStoreInstallResult({
      items: [
        { id: 'alice/a', status: 'installed' },
        { id: 'alice/b', status: 'failed' },
        { id: 'alice/c', status: 'skipped' },
      ],
    })
  ).toEqual({
    installed: 1,
    skipped: 1,
    failed: 1,
    failedItems: ['alice/b'],
  });

  expect(buildExtensionAssetUrl('alice/demo', 'assets/icon.png', 'abc 123')).toBe(
    '/sd-api/v2/extension/asset?id=alice%2Fdemo&path=assets%2Ficon.png&token=abc+123'
  );
});
