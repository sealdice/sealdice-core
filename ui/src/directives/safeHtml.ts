import type { Directive } from 'vue';
import DOMPurify, { type Config as DOMPurifyConfig } from 'dompurify';

type SafeHtmlValue =
  | string
  | {
      html: string;
      config?: DOMPurifyConfig;
    };

const DEFAULT_CONFIG: DOMPurifyConfig = {
  ALLOWED_TAGS: [
    'a',
    'b',
    'blockquote',
    'br',
    'code',
    'div',
    'em',
    'i',
    'li',
    'ol',
    'p',
    'pre',
    'span',
    'strong',
    'u',
    'ul',
  ],
  ALLOWED_ATTR: ['href', 'target', 'rel'],
};

function applySafeHtml(el: HTMLElement, value: SafeHtmlValue): void {
  if (!value) {
    el.innerHTML = '';
    return;
  }

  const html = typeof value === 'string' ? value : value.html;
  const config = typeof value === 'string' ? DEFAULT_CONFIG : { ...DEFAULT_CONFIG, ...value.config };
  el.innerHTML = DOMPurify.sanitize(html, config);
}

const safeHtmlDirective: Directive<HTMLElement, SafeHtmlValue> = {
  beforeMount(el, binding) {
    applySafeHtml(el, binding.value);
  },
  updated(el, binding) {
    applySafeHtml(el, binding.value);
  },
};

export default safeHtmlDirective;
