import { shouldSuppressRollupWarning } from './warningPolicy';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(
  shouldSuppressRollupWarning({
    code: 'INVALID_ANNOTATION',
    id: '/project/node_modules/.pnpm/@vueuse+core@14.3.0/node_modules/@vueuse/core/dist/index.js',
    message: 'A comment "/* #__PURE__ */" contains an annotation that Rollup cannot interpret.',
  }),
  true,
);

assertEqual(
  shouldSuppressRollupWarning({
    code: 'EVAL',
    id: '/project/node_modules/.pnpm/eruda@3.4.3/node_modules/eruda/eruda.js',
    message: 'Use of eval in "node_modules/.pnpm/eruda@3.4.3/node_modules/eruda/eruda.js" is strongly discouraged.',
  }),
  true,
);

assertEqual(
  shouldSuppressRollupWarning({
    code: 'EVAL',
    id: '/project/src/features/debug/local.ts',
    message: 'Use of eval in "src/features/debug/local.ts" is strongly discouraged.',
  }),
  false,
);

assertEqual(
  shouldSuppressRollupWarning({
    code: 'INVALID_ANNOTATION',
    id: '/project/node_modules/.pnpm/other/node_modules/other/index.js',
    message: 'A comment "/* #__PURE__ */" contains an annotation that Rollup cannot interpret.',
  }),
  false,
);
