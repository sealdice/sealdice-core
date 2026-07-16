import { routeMeta } from './routeMeta.ts';
import { appNavigation } from './navigation.ts';
import { buildRouteMeta } from './navigationModel.ts';
import { it } from 'vitest';

it('passes', async () => {

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(routeMeta['/mod/reply']?.layout, 'workspace');
assertEqual(routeMeta['/mod/deck']?.layout, 'wide');
assertEqual(routeMeta['/mod/story']?.layout, 'workspace');
assertEqual(routeMeta['/mod/js']?.layout, 'workspace');

assertEqual(routeMeta['/']?.layout, 'default');
assertEqual(routeMeta['/connect']?.layout, 'default');
assertEqual(routeMeta['/about']?.layout, 'wide');

const expectedRouteMeta = buildRouteMeta(appNavigation);
const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(routeMeta, expectedRouteMeta);
});
