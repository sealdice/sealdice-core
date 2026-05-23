import { joinApiBasePath, resolveApiBaseUrlFromLocation } from './config';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/v2ui/',
}), 'https://example.test');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/v2ui/',
}), 'https://example.test/dice');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/nested/v2ui/mod/story',
}), 'https://example.test/dice/nested');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/dice/v2ui-assets/',
}), 'https://example.test');

assertEqual(resolveApiBaseUrlFromLocation({
  origin: 'https://example.test',
  pathname: '/',
}), 'https://example.test');

assertEqual(joinApiBasePath('https://example.test/dice', '/sd-api/v2/base/health'), 'https://example.test/dice/sd-api/v2/base/health');
assertEqual(joinApiBasePath('https://example.test/dice/', 'sd-api/v2/realtime/ws'), 'https://example.test/dice/sd-api/v2/realtime/ws');
