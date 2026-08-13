import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { it } from 'vitest';

it('passes', async () => {

const rootDir = fileURLToPath(new URL('../..', import.meta.url));

const packageJson = JSON.parse(readFileSync(resolve(rootDir, 'package.json'), 'utf8')) as {
  scripts: Record<string, string>;
};
const openApiConfig = readFileSync(resolve(rootDir, 'openapi-ts.config.ts'), 'utf8');

const assertIncludes = (actual: string, expected: string, label: string) => {
  if (!actual.includes(expected)) {
    throw new Error(`${label} should include ${expected}, got ${actual}`);
  }
};

const assertNotIncludes = (actual: string, unexpected: string, label: string) => {
  if (actual.includes(unexpected)) {
    throw new Error(`${label} should not include ${unexpected}`);
  }
};

assertIncludes(packageJson.scripts.build, 'generate-api', 'build script');
assertIncludes(packageJson.scripts['build:embed'], 'generate-api', 'build:embed script');
assertNotIncludes(openApiConfig, 'DEFAULT_OPENAPI_SERVER', 'openapi-ts config');
assertNotIncludes(openApiConfig, 'VITE_API_PROXY_TARGET', 'openapi-ts config');
});
