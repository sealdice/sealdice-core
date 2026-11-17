import { resolveHashHistoryBase } from './historyBase';
import { expect, it } from 'vitest';

it('Hash 路由在相对部署路径下解析基址', async () => {
  expect(resolveHashHistoryBase('./')).toBe(undefined);
  expect(resolveHashHistoryBase('')).toBe(undefined);
  expect(resolveHashHistoryBase('/fixed/')).toBe('/fixed/');
});
