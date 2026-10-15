import { resolveViteBuildOptions, resolveVitePublicBase } from './embedConfig';
import { expect, it } from 'vitest';

it('内嵌构建输出使用相对资源路径', async () => {
  expect(resolveVitePublicBase('development')).toEqual('./');
  expect(resolveVitePublicBase('production')).toEqual('./');
  expect(resolveVitePublicBase('embed')).toEqual('./');

  expect(resolveViteBuildOptions('production')).toEqual({});
  expect(resolveViteBuildOptions('embed')).toEqual({
    outDir: '../static/v2ui/dist',
    emptyOutDir: true,
  });
});
