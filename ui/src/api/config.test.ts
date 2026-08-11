import { joinApiBasePath, resolveApiBaseUrlFromLocation, resolveOldUIUrlFromLocation } from './config';
import { it } from 'vitest';

it('passes', async () => {

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/',
}), 'https://example.test');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/',
}), 'https://example.test/dice');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice',
}), 'https://example.test/dice');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/index.html',
}), 'https://example.test/dice');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/index.html',
}), 'https://example.test');

assertEqual(resolveOldUIUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/',
}), 'https://example.test/old-ui/');

assertEqual(resolveOldUIUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/',
}), 'https://example.test/dice/old-ui/');

assertEqual(joinApiBasePath('https://example.test/dice', '/sd-api/v2/base/health'), 'https://example.test/dice/sd-api/v2/base/health');
assertEqual(joinApiBasePath('https://example.test/dice/', 'sd-api/v2/realtime/sse'), 'https://example.test/dice/sd-api/v2/realtime/sse');
});
