import { resolveViteBuildOptions, resolveVitePublicBase } from './embedConfig';

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertDeepEqual(resolveVitePublicBase('development'), './');
assertDeepEqual(resolveVitePublicBase('production'), './');
assertDeepEqual(resolveVitePublicBase('embed'), './');

assertDeepEqual(resolveViteBuildOptions('production'), {});
assertDeepEqual(resolveViteBuildOptions('embed'), {
  outDir: '../static/v2ui/dist',
  emptyOutDir: true,
});
