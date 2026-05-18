import { defineConfig } from '@hey-api/openapi-ts';
import { existsSync } from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

const configDir = path.dirname(fileURLToPath(import.meta.url));
const localSpecPath = path.join(configDir, 'openapi.json');
const DEFAULT_OPENAPI_SERVER = 'http://127.0.0.1:3005';
const openApiServer = (
  process.env.VITE_API_PROXY_TARGET ||
  process.env.DEV_PROXY_SERVER ||
  process.env.VITE_API_BASE_URL ||
  DEFAULT_OPENAPI_SERVER
).replace(/\/+$/, '');

export default defineConfig({
  input: existsSync(localSpecPath) ? localSpecPath : `${openApiServer}/openapi.json`,
  output: {
    path: path.join(configDir, 'src/api/generated'),
    postProcess: ['prettier', 'eslint'],
  },
  plugins: [
    {
      name: '@hey-api/client-fetch',
      baseUrl: false,
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
