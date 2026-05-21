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
  classifyVendorChunk('/project/node_modules/.pnpm/@vueuse+core@14.3.0/node_modules/@vueuse/core/dist/index.js'),
  'vendor-utility',
);
assertEqual(
  classifyVendorChunk('/project/node_modules/.pnpm/pinyin-pro@3.28.1/node_modules/pinyin-pro/dist/index.mjs'),
  'vendor-utility',
);
assertEqual(classifyVendorChunk('/project/src/main.ts'), undefined);
