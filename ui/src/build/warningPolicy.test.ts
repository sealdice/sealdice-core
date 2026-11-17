import { shouldSuppressRollupWarning } from './warningPolicy';
import { expect, it } from 'vitest';

it('仅忽略已知且可解释的构建警告', async () => {
  expect(
    shouldSuppressRollupWarning({
      code: 'INVALID_ANNOTATION',
      id: '/project/node_modules/.pnpm/@vueuse+core@14.3.0/node_modules/@vueuse/core/dist/index.js',
      message: 'A comment "/* #__PURE__ */" contains an annotation that Rollup cannot interpret.',
    })
  ).toBe(true);

  expect(
    shouldSuppressRollupWarning({
      code: 'EVAL',
      id: '/project/node_modules/.pnpm/eruda@3.4.3/node_modules/eruda/eruda.js',
      message:
        'Use of eval in "node_modules/.pnpm/eruda@3.4.3/node_modules/eruda/eruda.js" is strongly discouraged.',
    })
  ).toBe(true);

  expect(
    shouldSuppressRollupWarning({
      code: 'INVALID_ANNOTATION',
      id: '/project/node_modules/.pnpm/fzstd@0.1.1/node_modules/fzstd/esm/index.mjs',
      message:
        'A comment "/*#__PURE__*/" contains an annotation that Rolldown cannot interpret due to the position of the comment.',
    })
  ).toBe(true);

  expect(
    shouldSuppressRollupWarning({
      code: 'EVAL',
      id: '/project/src/features/debug/local.ts',
      message: 'Use of eval in "src/features/debug/local.ts" is strongly discouraged.',
    })
  ).toBe(false);

  expect(
    shouldSuppressRollupWarning({
      code: 'INVALID_ANNOTATION',
      id: '/project/node_modules/.pnpm/other/node_modules/other/index.js',
      message: 'A comment "/* #__PURE__ */" contains an annotation that Rollup cannot interpret.',
    })
  ).toBe(false);
});
