import { joinApiBasePath, resolveApiBaseUrlFromLocation, resolveSwitchToOldUIUrl } from './config';
import { expect, it } from 'vitest';

it('API 与旧界面切换使用同源地址，保留服务端偏好切换入口', () => {
  const location = { origin: 'https://example.test/', pathname: '/dice/v2ui/' };
  expect(resolveApiBaseUrlFromLocation(location)).toBe('https://example.test');
  expect(resolveSwitchToOldUIUrl(location)).toBe('https://example.test/dice/?ui=old');
  expect(
    resolveSwitchToOldUIUrl({ origin: 'https://example.test', pathname: '/dice/nested/v2ui/' })
  ).toBe('https://example.test/dice/nested/?ui=old');
  expect(joinApiBasePath('https://example.test/dice', '/sd-api/v2/base/health')).toBe(
    'https://example.test/dice/sd-api/v2/base/health'
  );
  expect(joinApiBasePath('https://example.test/dice/', 'sd-api/v2/realtime/sse')).toBe(
    'https://example.test/dice/sd-api/v2/realtime/sse'
  );
});
