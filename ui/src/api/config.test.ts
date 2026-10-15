import {
  joinApiBasePath,
  resolveApiBaseUrlFromLocation,
  resolveOldUIUrlFromLocation,
} from './config';
import { expect, it } from 'vitest';

it('API 地址与旧界面跳转按同源路径解析', async () => {
  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/',
    })
  ).toBe('https://example.test');

  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/dice/',
    })
  ).toBe('https://example.test');

  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/dice',
    })
  ).toBe('https://example.test');

  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/connect',
    })
  ).toBe('https://example.test');

  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/dice/index.html',
    })
  ).toBe('https://example.test');

  expect(
    resolveApiBaseUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/index.html',
    })
  ).toBe('https://example.test');

  expect(
    resolveOldUIUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/',
    })
  ).toBe('https://example.test/old-ui/');

  expect(
    resolveOldUIUrlFromLocation({
      origin: 'https://example.test',
      pathname: '/dice/',
    })
  ).toBe('https://example.test/old-ui/');

  expect(joinApiBasePath('https://example.test/dice', '/sd-api/v2/base/health')).toBe(
    'https://example.test/dice/sd-api/v2/base/health'
  );
  expect(joinApiBasePath('https://example.test/dice/', 'sd-api/v2/realtime/sse')).toBe(
    'https://example.test/dice/sd-api/v2/realtime/sse'
  );
});
