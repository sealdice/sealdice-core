import { cloneSearchFormValues, overwriteSearchFormValues } from './viewModel.js';
import { expect, it } from 'vitest';

it('搜索表单克隆不共享可变嵌套数据', async () => {
  const values = {
    keyword: 'story',
    platforms: ['QQ', 'Discord'],
    dateRange: [1710000000, 1710086400],
    nested: {
      includeArchived: true,
    },
  };

  const cloned = cloneSearchFormValues(values);
  expect(cloned).toEqual(values);
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

  expect(overwritten).toBe(target);
  expect(target).toEqual({
    keyword: 'new',
    platforms: ['QQ'],
  });

  const nextSource = {
    keyword: 'reply',
    platforms: ['QQ'],
  };

  overwriteSearchFormValues(target, nextSource);
  nextSource.platforms.push('Telegram');
  expect(target.platforms).toEqual(['QQ']);
});
