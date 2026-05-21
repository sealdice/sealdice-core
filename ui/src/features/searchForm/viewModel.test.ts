import {
  cloneSearchFormValues,
  overwriteSearchFormValues,
} from './viewModel.js';

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
};

const assertDeepEqual = (actual: unknown, expected: unknown) => {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

const values = {
  keyword: 'story',
  platforms: ['QQ', 'Discord'],
  dateRange: [1710000000, 1710086400],
  nested: {
    includeArchived: true,
  },
};

const cloned = cloneSearchFormValues(values);
assertDeepEqual(cloned, values);
if (cloned === values) {
  throw new Error('cloneSearchFormValues should return a new object');
}
if (cloned.platforms === values.platforms) {
  throw new Error('cloneSearchFormValues should clone arrays');
}
if (cloned.nested === values.nested) {
  throw new Error('cloneSearchFormValues should clone nested objects');
}

const target = {
  keyword: 'old',
  platforms: ['KOOK'],
  stale: true,
};

const overwritten = overwriteSearchFormValues(target, {
  keyword: 'new',
  platforms: ['QQ'],
});

assertEqual(overwritten, target);
assertDeepEqual(target, {
  keyword: 'new',
  platforms: ['QQ'],
});

const nextSource = {
  keyword: 'reply',
  platforms: ['QQ'],
};

overwriteSearchFormValues(target, nextSource);
nextSource.platforms.push('Telegram');
assertDeepEqual(target.platforms, ['QQ']);
