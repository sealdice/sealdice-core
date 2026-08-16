import { readdirSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createJiti } from 'jiti';

const rootDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const srcDir = join(rootDir, 'src');
const testFiles = [];

function collectTestFiles(dir) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      collectTestFiles(path);
    } else if (entry.endsWith('.test.ts')) {
      testFiles.push(path);
    }
  }
}

collectTestFiles(srcDir);
testFiles.sort();

const jiti = createJiti(import.meta.url, {
  alias: {
    '@': srcDir,
  },
});

for (const file of testFiles) {
  await jiti.import(pathToFileURL(file).href);
  console.log(`PASS ${relative(rootDir, file)}`);
}

console.log(`${testFiles.length} test files passed`);
