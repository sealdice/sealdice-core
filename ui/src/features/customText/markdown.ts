import MarkdownIt from 'markdown-it';
import katex from 'katex';
import texmath from 'markdown-it-texmath';
import DOMPurify from 'dompurify';

const markdown = new MarkdownIt({ html: false, breaks: true, linkify: true })
  .disable('table')
  .use(texmath, {
    engine: katex,
    delimiters: 'dollars',
    katexOptions: {
      trust: false,
      strict: 'ignore',
      throwOnError: false,
      maxExpand: 1000,
      maxSize: 20,
    },
  });
const defaultStrong = markdown.renderer.rules['strong_open'];
markdown.renderer.rules['strong_open'] = (tokens, index, options, environment, renderer) => {
  if (tokens[index].markup === '__') tokens[index].attrJoin('class', 'qq-underline-bold');
  return defaultStrong
    ? defaultStrong(tokens, index, options, environment, renderer)
    : renderer.renderToken(tokens, index, options);
};
markdown.renderer.rules['image'] = (tokens, index, options, environment, renderer) => {
  const token = tokens[index];
  const description = renderer.renderInlineAsText(token.children ?? [], options, environment);
  const dimensions = /^(.*?)\s*#([1-9]\d*)px\s+#([1-9]\d*)px\s*$/is.exec(description);
  const source = token.attrGet('src') ?? '';
  const escape = markdown.utils.escapeHtml;
  let alt = description;
  let sizing = '';
  if (dimensions) {
    const width = Number(dimensions[2]);
    const height = Number(dimensions[3]);
    if (Number.isSafeInteger(width) && Number.isSafeInteger(height)) {
      alt = dimensions[1].trimEnd();
      sizing = ` data-image-width="${width}" data-image-height="${height}" style="width:${width}px;aspect-ratio:${width}/${height}"`;
    }
  }
  // An inert URL in data-* cannot start a request when the preview is inserted.
  return `<button type="button" class="image-placeholder" data-image-src="${escape(source)}" data-image-alt="${escape(alt)}" aria-label="${escape(alt ? `加载图片：${alt}` : '加载图片')}" title="${escape(`点击后将请求图片地址：${source}`)}"${sizing}><span>图片占位 · 点击加载</span>${alt ? `<span>${escape(alt)}</span>` : ''}</button>`;
};
const defaultLink = markdown.renderer.rules['link_open'];
markdown.renderer.rules['link_open'] = (tokens, index, options, environment, renderer) => {
  tokens[index].attrSet('target', '_blank');
  tokens[index].attrSet('rel', 'noopener noreferrer');
  return defaultLink
    ? defaultLink(tokens, index, options, environment, renderer)
    : renderer.renderToken(tokens, index, options);
};
export function renderMarkdown(source: string): string {
  return DOMPurify.sanitize(`<div>${markdown.render(source)}</div>`, { ADD_ATTR: ['target'] });
}

const plainTokens = new Set([
  'paragraph_open',
  'paragraph_close',
  'text',
  'text_special',
  'softbreak',
  'hardbreak',
]);
function containsFormatting(tokens: ReturnType<typeof markdown.parse>): boolean {
  return tokens.some(token =>
    token.type === 'inline'
      ? containsFormatting(token.children ?? [])
      : !plainTokens.has(token.type)
  );
}

/** Inspect evaluated text using the same Markdown rules; plain paragraphs need no duplicate preview. */
export function renderMarkdownIfFormatted(source: string): string | undefined {
  const environment = {};
  const tokens = markdown.parse(source, environment);
  if (!containsFormatting(tokens)) return undefined;
  return DOMPurify.sanitize(
    `<div>${markdown.renderer.render(tokens, markdown.options, environment)}</div>`,
    { ADD_ATTR: ['target'] }
  );
}
