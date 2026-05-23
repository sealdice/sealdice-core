import { classifyVendorChunk } from './chunkPolicy';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/vue@3.5.34/node_modules/vue/dist/vue.runtime.esm-bundler.js'),
  'vendor-framework',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/naive-ui@2.43.1/node_modules/naive-ui/es/button/index.mjs'),
  undefined,
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/pro-naive-ui@3.2.3/node_modules/pro-naive-ui/es/index.js'),
  undefined,
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/vueuc@0.4.65/node_modules/vueuc/es/shared/index.js'),
  undefined,
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/css-render@0.15.14/node_modules/css-render/esm/index.js'),
  undefined,
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/eruda@3.4.3/node_modules/eruda/eruda.js'),
  'vendor-debug',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/lodash-es@4.17.21/node_modules/lodash-es/merge.js'),
  'vendor-lodash',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/dayjs@1.11.18/node_modules/dayjs/dayjs.min.js'),
  'vendor-date',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/date-fns@3.6.0/node_modules/date-fns/locale/zh-CN.mjs'),
  'vendor-date',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/@vueuse+core@14.3.0/node_modules/@vueuse/core/dist/index.js'),
  'vendor-utility',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/pinyin-pro@3.28.1/node_modules/pinyin-pro/dist/index.mjs'),
  'vendor-utility',
);
assertEqual(classifyVendorChunk('/project/src/main.ts'), undefined);
