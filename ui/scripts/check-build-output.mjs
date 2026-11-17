import assert from 'node:assert/strict';
import { readFileSync, existsSync, statSync } from 'node:fs';
import { resolve, relative } from 'node:path';
import ts from 'typescript';

const root = resolve(process.argv[2] || '../static/v2ui/dist');
const manifest = JSON.parse(readFileSync(resolve(root, '.vite/manifest.json'), 'utf8'));
assert(
  Object.values(manifest).some(chunk => chunk.isEntry),
  'missing entry in build manifest'
);
for (const chunk of Object.values(manifest)) {
  assert(existsSync(resolve(root, chunk.file)), `missing output: ${chunk.file}`);
  for (const imported of [...(chunk.imports || []), ...(chunk.dynamicImports || [])]) {
    assert(manifest[imported], `missing chunk reference: ${imported}`);
  }
}
const pwa = JSON.parse(readFileSync(resolve(root, 'manifest.webmanifest'), 'utf8'));
assert(
  pwa.icons.some(icon => icon.sizes === '192x192'),
  'missing install icon'
);
assert(
  pwa.icons.some(icon => icon.purpose?.includes('maskable')),
  'missing maskable icon'
);
for (const icon of pwa.icons)
  assert(existsSync(resolve(root, icon.src)), `missing icon: ${icon.src}`);
// Workbox 在 manifestTransforms 之后还会加入图标等附加资源；核对最终 SW
// 的实际预缓存表，避免只统计前一阶段的清单。不执行生成的脚本。
const worker = ts.createSourceFile(
  'sw.js',
  readFileSync(resolve(root, 'sw.js'), 'utf8'),
  ts.ScriptTarget.Latest,
  true,
  ts.ScriptKind.JS
);
let precache;
function inspect(node) {
  if (
    ts.isCallExpression(node) &&
    ts.isPropertyAccessExpression(node.expression) &&
    node.expression.name.text === 'precacheAndRoute'
  ) {
    assert(!precache, 'ambiguous precache registration');
    assert(ts.isArrayLiteralExpression(node.arguments[0]), 'precache must be a literal manifest');
    precache = node.arguments[0].elements.map(entry => {
      assert(ts.isObjectLiteralExpression(entry), 'invalid precache entry');
      const url = entry.properties.find(
        prop =>
          ts.isPropertyAssignment(prop) &&
          prop.name.getText(worker).replaceAll('"', '').replaceAll("'", '') === 'url'
      );
      assert(url && ts.isStringLiteral(url.initializer), 'invalid precache URL');
      return url.initializer.text;
    });
  }
  ts.forEachChild(node, inspect);
}
inspect(worker);
assert(precache?.includes('index.html'), 'missing offline app shell');
assert(
  !precache.some(url => /(?:^|\/)vendor-debug-[^/]+\.js$/.test(url)),
  'optional debug tool is precached'
);
const bytes = precache.reduce((sum, url) => {
  const file = resolve(root, decodeURIComponent(url));
  assert(!relative(root, file).startsWith('..'), 'precache path escapes build directory');
  return sum + statSync(file).size;
}, 0);
const budget = JSON.parse(readFileSync(resolve(root, '.vite/resource-budget.json'), 'utf8'));
assert(
  bytes <= budget.precacheBytes,
  `final PWA precache ${bytes} bytes exceeds budget ${budget.precacheBytes}`
);
console.log(`Final PWA precache: ${bytes} bytes, ${precache.length} entries`);
console.log('Actual build manifest, lazy chunk references and PWA assets verified.');
