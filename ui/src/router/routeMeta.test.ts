import { routeMeta } from './routeMeta.ts';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
};

assertEqual(routeMeta['/mod/reply']?.layout, 'wide');
assertEqual(routeMeta['/mod/deck']?.layout, 'wide');
assertEqual(routeMeta['/mod/story']?.layout, 'wide');
assertEqual(routeMeta['/mod/js']?.layout, 'wide');

assertEqual(routeMeta['/']?.layout, 'default');
assertEqual(routeMeta['/connect']?.layout, 'default');
