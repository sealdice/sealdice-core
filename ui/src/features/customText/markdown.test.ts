// @vitest-environment happy-dom
import { expect, it } from 'vitest';
import { renderMarkdown, renderMarkdownIfFormatted } from './markdown';

it('renders QQ emphasis, inline/fenced code and display math without tables', () => {
  const html = renderMarkdown(
    '**加粗** __下划线__ `行内代码`\n\n```js\nconst x = 1;\n```\n\n$$x^2$$\n\n| A | B |\n|---|---|\n| 1 | 2 |'
  );
  expect(html).toContain('<strong>加粗</strong>');
  expect(html).toContain('class="qq-underline-bold"');
  expect(html).toContain('<code>行内代码</code>');
  expect(html).toContain('language-js');
  expect(html).toContain('katex-display');
  expect(html).not.toContain('<table');
});

it.each([
  '',
  '侦查=42/60 成功',
  '第一行\n第二行\n\n第三行',
  'a_name_with_underscores',
  '\\*普通星号\\*',
  '数值 1 < 2，2 > 1',
  '| A | B |\n|---|---|\n| 1 | 2 |',
])('does not duplicate ordinary text: %s', source => {
  expect(renderMarkdownIfFormatted(source)).toBeUndefined();
});

it.each([
  ['**加粗**', 'strong'],
  ['__下划线__', 'strong.qq-underline-bold'],
  ['*斜体*', 'em'],
  ['~~删除线~~', 's'],
  ['# 标题', 'h1'],
  ['> 引用', 'blockquote'],
  ['- 列表', 'ul > li'],
  ['1. 列表', 'ol > li'],
  ['[链接](https://example.com)', 'a[href="https://example.com"]'],
  ['https://example.com', 'a[href="https://example.com"]'],
  [
    '![图片 #320px #200px](https://example.com/a.png)',
    'button[data-image-width="320"][data-image-height="200"]',
  ],
  ['`行内代码`', 'code'],
  ['```\n代码\n```', 'pre > code'],
  ['$$x^2$$', '.katex-display'],
  ['---', 'hr'],
])('renders the expected formatting using the actual preview grammar: %s', (source, selector) => {
  const output = document.createElement('div');
  output.innerHTML = renderMarkdownIfFormatted(source) ?? '';
  expect(output.querySelector(selector)).not.toBeNull();
});

it('preserves QQ image dimensions and responsive aspect ratio', () => {
  const html = renderMarkdown('![壁纸 #320px #200px](https://example.com/test.png)');
  expect(html).toContain('data-image-alt="壁纸"');
  expect(html).toContain('data-image-width="320"');
  expect(html).toContain('data-image-height="200"');
  expect(html).toContain('aspect-ratio:320/200');
});

it.each([renderMarkdown, renderMarkdownIfFormatted])(
  'renders inert, escaped image placeholders',
  render => {
    const output = document.createElement('div');
    output.innerHTML =
      render('![<img src=x onerror=alert(1)>](https://example.com/a.png?a=1&b=2)') ?? '';
    expect(output.querySelector('img, [src], [srcset], [onerror]')).toBeNull();
    const placeholder = output.querySelector<HTMLButtonElement>('button.image-placeholder')!;
    expect(placeholder.type).toBe('button');
    expect(placeholder.dataset.imageSrc).toBe('https://example.com/a.png?a=1&b=2');
    expect(placeholder.textContent).toContain('<img src=x onerror=alert(1)>');
  }
);

it('escapes HTML and rejects active links and untrusted TeX commands', () => {
  const html = renderMarkdown(
    '<script>alert(1)</script>\n[x](javascript:alert(1))\n$$\\href{javascript:alert(1)}{x}$$'
  );
  const element = document.createElement('div');
  element.innerHTML = html;
  expect(element.querySelector('script, iframe, [onclick]')).toBeNull();
  expect(element.querySelector('a[href^="javascript:"]')).toBeNull();
  expect(element.textContent).toContain('<script>');
});
