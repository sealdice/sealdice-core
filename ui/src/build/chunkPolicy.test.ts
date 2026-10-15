import { expect, it } from 'vitest';
import { classifyVendorChunk } from './chunkPolicy';

it.each([
  ['@uppy+core@6.2.0/node_modules/@uppy/core/lib/Uppy.js', 'vendor-upload'],
  ['tus-js-client@4.3.1/node_modules/tus-js-client/lib/upload.js', 'vendor-upload'],
  ['vue@3.5.34/node_modules/vue/dist/vue.runtime.esm-bundler.js', 'vendor-framework'],
  ['naive-ui@2.43.1/node_modules/naive-ui/es/button/index.mjs', undefined],
  ['pro-naive-ui@3.2.3/node_modules/pro-naive-ui/es/index.js', undefined],
  ['vueuc@0.4.65/node_modules/vueuc/es/shared/index.js', undefined],
  ['css-render@0.15.14/node_modules/css-render/esm/index.js', undefined],
  ['eruda@3.4.3/node_modules/eruda/eruda.js', 'vendor-debug'],
  ['lodash-es@4.17.21/node_modules/lodash-es/merge.js', 'vendor-lodash'],
  ['dayjs@1.11.18/node_modules/dayjs/dayjs.min.js', 'vendor-date'],
  ['date-fns@3.6.0/node_modules/date-fns/locale/zh-CN.mjs', 'vendor-date'],
  ['@vueuse+core@14.3.0/node_modules/@vueuse/core/dist/index.js', 'vendor-utility'],
  ['pinyin-pro@3.28.1/node_modules/pinyin-pro/dist/index.mjs', 'vendor-utility'],
])('classifies the pnpm module %s', (module, expected) => {
  expect(classifyVendorChunk(`/project/node_modules/.pnpm/${module}`)).toBe(expected);
});

it('leaves application modules to the automatic chunk graph', () => {
  expect(classifyVendorChunk('/project/src/main.ts')).toBeUndefined();
});
