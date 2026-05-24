import { defineConfig } from '@hey-api/openapi-ts';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const configDir = path.dirname(fileURLToPath(import.meta.url));
const localSpecPath = path.join(configDir, 'openapi.json');

export default defineConfig({
  input: localSpecPath,
  output: {
    path: path.join(configDir, 'src/api/generated'),
    postProcess: ['prettier', 'eslint'],
  },
  plugins: [
    {
      name: '@hey-api/client-axios',
      baseURL: false,
      bundle: true,
    },
    '@hey-api/typescript',
    '@hey-api/schemas',
    {
      name: '@hey-api/sdk',
      operations: {
        strategy: 'flat',
        nesting: 'operationId',
        nestingDelimiters: /[-./]/,
      },
    },
    {
      name: '@tanstack/vue-query',
      exportFromIndex: true,
      queryKeys: true,
      queryOptions: true,
      infiniteQueryOptions: false,
    },
  ],
});
