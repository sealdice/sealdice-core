import {
  buildCustomTextExportContent,
  createTextItemKeyStore,
  getCustomTextGroups,
  parseCustomTextImportContent,
  sortCustomTextCategory,
} from './viewModel';
import type { TextTemplateHelpDict, TextTemplateItem, TextTemplateWithWeightDict } from './types';
import { expect, it } from 'vitest';

it('自定义文案的权重与分类转换', async () => {
  const assertOk = (actual: unknown) => {
    if (!actual) {
      throw new Error(`expected truthy value, got ${String(actual)}`);
    }
  };

  const texts: TextTemplateWithWeightDict = {
    core: {
      diceName: [['SealDice', 1]],
      deprecated: [['old', 1]],
      modified: [['new', 1]],
    },
  };

  const helpInfo: TextTemplateHelpDict = {
    core: {
      diceName: {
        commands: null,
        exampleCommands: null,
        extraText: '',
        filename: null,
        modified: false,
        notBuiltin: false,
        origin: null,
        subType: '基础 名称',
        topOrder: 0,
        vars: ['$t玩家'],
      },
      deprecated: {
        commands: null,
        exampleCommands: null,
        extraText: '',
        filename: null,
        modified: false,
        notBuiltin: true,
        origin: null,
        subType: '旧版 条目',
        topOrder: 0,
        vars: null,
      },
      modified: {
        commands: null,
        exampleCommands: null,
        extraText: '',
        filename: null,
        modified: true,
        notBuiltin: false,
        origin: null,
        subType: '基础 修改',
        topOrder: 10,
        vars: null,
      },
    },
  };

  expect(getCustomTextGroups(helpInfo.core)).toEqual(['基础', '旧版']);
  expect(
    sortCustomTextCategory({ texts, helpInfo, category: 'core', filterMode: 'modified' })[0]![1]
      .length
  ).toBe(1);
  expect(
    sortCustomTextCategory({
      texts,
      helpInfo,
      category: 'core',
      filterMode: 'deprecated',
    })[0]![1][0]![0]
  ).toBe('deprecated');

  const compact = buildCustomTextExportContent({
    texts,
    category: 'core',
    onlyCurrent: true,
    compact: true,
  });
  expect(JSON.parse(compact).items.core.diceName[0][0]).toBe('SealDice');

  const pretty = buildCustomTextExportContent({
    texts,
    category: 'core',
    onlyCurrent: false,
    compact: false,
  });
  assertOk(pretty.includes('\n  '));

  const parsed = parseCustomTextImportContent(JSON.stringify({ title: 'x', items: texts }));
  expect(parsed.core.diceName).toEqual([['SealDice', 1]]);

  const keyStore = createTextItemKeyStore();
  const item: TextTemplateItem = ['text', 1];
  expect(keyStore.keyOf('diceName', item)).toBe(keyStore.keyOf('diceName', item));
});
