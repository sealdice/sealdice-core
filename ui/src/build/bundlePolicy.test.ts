/// <reference types="node" />

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import assert from 'node:assert/strict';

const mainEntry = readFileSync(resolve(process.cwd(), 'src/main.ts'), 'utf8');

assert.equal(
  mainEntry.includes('app.use(proNaiveUi)'),
  false,
  'Do not install pro-naive-ui globally; it pulls the full component library into the entry chunk.',
);
