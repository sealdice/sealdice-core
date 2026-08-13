import {
  buildExtensionAssetUrl,
  buildPackageFileTree,
  summarizeStoreInstallResult,
} from './model.js';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  const assertDeepEqual = (actual: unknown, expected: unknown) => {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
      throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
    }
  };

  assertDeepEqual(
    buildPackageFileTree(['reply/main.yaml', 'reply/assets/icon.png', 'scripts/index.js']),
    [
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
    ]
  );

  assertDeepEqual(
    summarizeStoreInstallResult({
      items: [
        { id: 'alice/a', status: 'installed' },
        { id: 'alice/b', status: 'failed' },
        { id: 'alice/c', status: 'skipped' },
      ],
    }),
    {
      installed: 1,
      skipped: 1,
      failed: 1,
      failedItems: ['alice/b'],
    }
  );

  assertEqual(
    buildExtensionAssetUrl('alice/demo', 'assets/icon.png', 'abc 123'),
    '/sd-api/v2/extension/asset?id=alice%2Fdemo&path=assets%2Ficon.png&token=abc+123'
  );
});
