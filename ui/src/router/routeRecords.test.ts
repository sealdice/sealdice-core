import { withRouteMeta } from './routeRecords.ts';
import type { RouteRecordRaw } from 'vue-router';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

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
    children: [
      { path: ':category', component: DummyRoute },
    ],
  },
] satisfies RouteRecordRaw[];

const merged = withRouteMeta(records, {
  '/tool/resource': { title: '资源管理', layout: 'wide' },
  '/tool/test': { title: '指令测试', layout: 'default' },
  '/custom-text/:category': { title: '自定义文案', layout: 'default' },
});

assertEqual(merged[0]?.children?.[0]?.path, 'resource');
assertEqual(merged[0]?.children?.[0]?.meta?.title, '资源管理');
assertEqual(merged[0]?.children?.[0]?.meta?.layout, 'wide');
assertEqual(merged[0]?.children?.[1]?.meta?.title, '指令测试');
assertEqual(typeof merged[1]?.children?.[0]?.props, 'function');
