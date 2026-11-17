import { withRouteMeta } from './routeRecords.ts';
import type { RouteRecordRaw } from 'vue-router';
import { expect, it } from 'vitest';

it('merges child route metadata while preserving relative paths and dynamic props', () => {
  const DummyRoute = { name: 'DummyRoute' };

  const records = [
    {
      path: '/tool',
      component: DummyRoute,
      children: [
        { path: 'resource', component: DummyRoute },
        { path: 'test', component: DummyRoute },
      ],
    },
    {
      path: '/custom-text',
      component: DummyRoute,
      children: [{ path: ':category', component: DummyRoute }],
    },
  ] satisfies RouteRecordRaw[];

  const merged = withRouteMeta(records, {
    '/tool/resource': { title: '资源管理', layout: 'wide' },
    '/tool/test': { title: '指令测试', layout: 'default' },
    '/mod/reply': { title: '自定义回复', layout: 'workspace' },
    '/custom-text/:category': { title: '自定义文案', layout: 'default' },
  });

  expect(merged[0]?.children?.[0]).toMatchObject({
    path: 'resource',
    meta: { title: '资源管理', layout: 'wide' },
  });
  expect(merged[0]?.children?.[1]?.meta?.title).toBe('指令测试');
  expect(merged[1]?.children?.[0]?.props).toBeTypeOf('function');
});
