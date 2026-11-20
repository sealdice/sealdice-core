import { describe, expect, it } from 'vitest';
import { templateHighlights } from './highlight';
import { scriptSpans, scriptTokens } from './syntax';

const formulaTemplate = [
  '{% $tColor = "green";',
  '$tText = `\\n$$\\boxed\\{\\overset\\{\\color\\{{$tColor}\\} {$tValue}\\}\\{{$tResult}\\}\\}$$`;',
  '$tText %}',
].join('\n');

const parts = (source: string, kind: string) =>
  templateHighlights(source)
    .filter(span => span.kind === kind)
    .map(span => source.slice(span.from, span.to));

describe('nested DiceScript template highlighting', () => {
  it('keeps evaluating braces inside math consistent with DiceScript template semantics', () => {
    expect(parts('$$\\boxed{42/60}$$', 'expression')).toEqual(['{42/60}']);
    expect(parts('$$x+{1+2}$$', 'expression')).toEqual(['{1+2}']);
    expect(parts('$$\\boxed\\{42/60\\}$$', 'expression')).toEqual([]);
    expect(parts('$$\\boxed\\{值 {$tValue}\\}$$', 'expression')).toEqual(['{$tValue}']);
  });
  it('limits script block backgrounds to both markers while retaining expression backgrounds', () => {
    const source =
      '{% $tText = `结果 {$tValue}`; if true { $tText = "示例"; }; $tText %} {$tOther}';
    expect(parts(source, 'expression')).toEqual(['{%', '%}', '{$tOther}']);
    expect(parts(source, 'interpolation')).toEqual(['{$tValue}']);
    expect(parts(source, 'delimiter')).toEqual(['{%', '{', '}', '%}', '{', '}']);
    expect(parts(source, 'variable')).toContain('$tText');
    expect(parts(source, 'keyword')).toContain('if');
  });
  it('keeps adjacent and unfinished script blocks free of whole-body backgrounds', () => {
    expect(parts('{% 1 %}{% 2 %}', 'expression')).toEqual(['{%', '%}', '{%', '%}']);
    for (const source of ['{%', '{% $tValue = 1', '{% $tText = `{$tValue']) {
      expect(parts(source, 'expression')).toEqual(['{%']);
      expect(parts(source, 'delimiter')[0]).toBe('{%');
    }
  });
  it('highlights escapes in regular and backtick strings without overlapping string colours', () => {
    const quoteEscape = String.fromCharCode(92, 34);
    const source =
      '{% a="' + '\\n\\t\\r\\f\\\\' + quoteEscape + "\"; b='\\''; c=`\\n\\{字面量\\} {$t玩家}` %}";
    expect(parts(source, 'escape')).toEqual([
      '\\n',
      '\\t',
      '\\r',
      '\\f',
      '\\\\',
      quoteEscape,
      "\\'",
      '\\n',
      '\\{',
      '\\}',
    ]);
    for (const escape of templateHighlights(source).filter(span => span.kind === 'escape')) {
      expect(
        templateHighlights(source).some(
          span => span.kind === 'string' && span.from < escape.to && span.to > escape.from
        )
      ).toBe(false);
    }
    expect(parts(source, 'interpolation')).toEqual(['{$t玩家}']);
  });
  it('does not double-count escaped backslashes or mark unknown TeX commands as escapes', () => {
    const source = '{% x=`\\\\n \\boxed\\{\\overset\\{值\\}\\} \\color\\{red\\}` %}';
    expect(parts(source, 'escape')).toEqual(['\\\\', '\\{', '\\{', '\\}', '\\}', '\\{', '\\}']);
    expect(parts(source, 'string').join('')).toContain('\\boxed');
    expect(parts(source, 'string').join('')).toContain('\\overset');
  });
  it('highlights escaped braces and newlines in a synthetic QQ formula template', () => {
    const source = formulaTemplate;
    expect(parts(source, 'escape')).toContain('\\n');
    expect(parts(source, 'escape')).toContain('\\{');
    expect(parts(source, 'escape')).toContain('\\}');
    expect(parts(source, 'interpolation')).toContain('{$tColor}');
  });
  it('highlights interpolated variables and expressions separately from literal template text', () => {
    const source = '{% $t文本 = `检定 {$tD100} / {力量 + 5}`; $t文本 %}';
    expect(parts(source, 'interpolation')).toEqual(['{$tD100}', '{力量 + 5}']);
    expect(parts(source, 'variable')).toContain('$tD100');
    expect(parts(source, 'variable')).toContain('力量');
    expect(parts(source, 'number')).toContain('5');
    expect(parts(source, 'string')).toContain('`检定 ');
    const variablePosition = source.indexOf('$tD100');
    expect(
      templateHighlights(source).some(
        span =>
          span.kind === 'string' && span.from <= variablePosition && span.to > variablePosition
      )
    ).toBe(false);
  });
  it('distinguishes escaped TeX braces from actual template delimiters in a synthetic template', () => {
    const source = formulaTemplate;
    const nested = parts(source, 'interpolation');
    expect(nested).toContain('{$tColor}');
    expect(nested).toContain('{$tValue}');
    expect(nested).toContain('{$tResult}');
    expect(nested.some(part => part.includes('\\overset'))).toBe(false);
    const latexBrace = source.indexOf('\\boxed\\{') + '\\boxed'.length + 1;
    expect(
      templateHighlights(source).some(span => span.kind === 'delimiter' && span.from === latexBrace)
    ).toBe(false);
  });
  it('handles nested backtick templates and quoted braces without ending the outer string early', () => {
    const source = '{% $t文本 = `结果：{1 ? `成功 {$t玩家}` : "}"}`; $t文本 %}';
    expect(scriptSpans(source)).toEqual([{ from: 0, to: source.length, closed: true }]);
    expect(
      scriptTokens(source)
        .filter(token => token.kind === 'string')
        .map(token => token.text)
    ).toEqual(['`结果：{1 ? `成功 {$t玩家}` : "}"}`']);
    expect(parts(source, 'interpolation')).toEqual(['{1 ? `成功 {$t玩家}` : "}"}', '{$t玩家}']);
    expect(parts(source, 'variable')).toContain('$t玩家');
  });
  it('keeps quoted literals, escapes and comments out of interpolation highlighting', () => {
    const source =
      '{% a="{$t普通字符串}"; b=\'{$t普通字符串2}\'; c=`\\{$t转义\\} {$t正常}`; // {$t注释}\n%}';
    expect(parts(source, 'interpolation')).toEqual(['{$t正常}']);
    expect(parts(source, 'variable')).not.toContain('$t普通字符串');
    expect(parts(source, 'variable')).not.toContain('$t转义');
    expect(parts(source, 'variable')).not.toContain('$t注释');
  });
  it('provides bounded, valid ranges for unfinished input during editing', () => {
    for (const source of ['{% x=`未完成 {$t玩家', '{% x=`{1 ? `更深 {$t玩家', '{"未闭合']) {
      const highlights = templateHighlights(source);
      expect(highlights.length).toBeGreaterThan(0);
      expect(
        highlights.every(span => span.from >= 0 && span.to > span.from && span.to <= source.length)
      ).toBe(true);
    }
    expect(parts('{% x=`未完成 {$t玩家', 'variable')).toContain('$t玩家');
  });
});
