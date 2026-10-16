import { buildPackageFileTree } from './model';
import { expect, it } from 'vitest';

it('扩展包文件按目录组织并排序', () => {
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
});
