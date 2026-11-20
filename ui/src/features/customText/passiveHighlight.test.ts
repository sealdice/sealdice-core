import { expect, it } from 'vitest';
import { passiveHighlights } from './passiveHighlight';

it('uses marker-only script backgrounds in the idle editor while preserving interpolation backgrounds', () => {
  const source = '{% $tText = `结果 {$tValue}`; $tText %}';
  const parts = passiveHighlights(source);
  expect(parts.map(part => part.text).join('')).toBe(source);
  expect(
    parts.filter(part => part.classes.includes('seal-expression')).map(part => part.text)
  ).toEqual(['{%', '%}']);
  expect(parts.find(part => part.text === '$tValue')?.classes).toContain('seal-interpolation');
  expect(parts.find(part => part.text === '$tText')?.classes).not.toContain('seal-expression');
});

it('flattens nested template highlights without duplicating or losing any source characters', () => {
  const source = '{% $t文本 = `\\n结果：{1 ? `成功 {$t玩家}` : "}"}`; $t文本 %}';
  const parts = passiveHighlights(source);
  expect(parts.map(part => part.text).join('')).toBe(source);
  expect(
    parts.some(
      part =>
        part.text === '$t玩家' &&
        part.classes.includes('seal-variable') &&
        part.classes.includes('seal-interpolation')
    )
  ).toBe(true);
  expect(parts.some(part => part.text === '\\n' && part.classes.includes('seal-escape'))).toBe(
    true
  );
});

it('preserves plain text and incomplete input as text rather than generating HTML', () => {
  for (const source of [
    '',
    '<img src=x onerror=alert(1)>',
    '{% x=`未闭合 {$t变量',
    '中文😀\n\ttext',
  ]) {
    expect(
      passiveHighlights(source)
        .map(part => part.text)
        .join('')
    ).toBe(source);
  }
});
