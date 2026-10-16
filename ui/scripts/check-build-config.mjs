import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { loadConfigFromFile } from 'vite';

const result = await loadConfigFromFile(
  { command: 'build', mode: 'embed' },
  fileURLToPath(new URL('../vite.config.ts', import.meta.url))
);
const name = result.config.build.rolldownOptions.output.codeSplitting.groups[0].name;
assert.equal(result.config.base, './');
assert.equal(result.config.build.outDir, '../static/v2ui/dist');
assert.equal(result.config.build.emptyOutDir, true);
assert.equal(name('/project/node_modules/vue/dist/vue.runtime.esm-bundler.js'), 'vendor-framework');
assert.equal(name('/project/src/main.ts'), undefined);
console.log('Actual Vite chunk configuration verified.');
