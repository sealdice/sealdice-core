export interface ScriptSpan {
  from: number;
  to: number;
  closed: boolean;
}

const maxNesting = 64;

function quotedEnd(
  source: string,
  start: number,
  nesting: number
): { to: number; closed: boolean } {
  if (nesting > maxNesting) return { to: source.length, closed: false };
  const quote = source[start];
  for (let i = start + 1; i < source.length; i++) {
    const char = source[i];
    if (char === '\\') {
      i++;
      continue;
    }
    if (char === quote) return { to: i + 1, closed: true };
    if (quote === '`' && char === '{') {
      const expression = expressionEnd(source, i, nesting + 1);
      if (!expression.closed) return expression;
      i = expression.to - 1;
    }
  }
  return { to: source.length, closed: false };
}

function expressionEnd(
  source: string,
  start: number,
  nesting: number
): { to: number; closed: boolean } {
  if (nesting > maxNesting) return { to: source.length, closed: false };
  let depth = 1;
  for (let i = start + 1; i < source.length; i++) {
    const char = source[i];
    if (char === '\\') {
      i++;
      continue;
    }
    if (char === '"' || char === "'" || char === '`') {
      const quoted = quotedEnd(source, i, nesting + 1);
      if (!quoted.closed) return quoted;
      i = quoted.to - 1;
      continue;
    }
    if (char === '/' && source[i + 1] === '/') {
      const newline = source.indexOf('\n', i + 2);
      if (newline < 0) break;
      i = newline;
      continue;
    }
    if (char === '{') depth++;
    if (char === '}' && --depth === 0) return { to: i + 1, closed: true };
  }
  return { to: source.length, closed: false };
}

/** Locate balanced template expressions, including nested braces and quoted template strings. */
export function scriptSpans(source: string): ScriptSpan[] {
  const spans: ScriptSpan[] = [];
  for (let i = 0; i < source.length; i++) {
    if (source[i] === '\\') {
      i++;
      continue;
    }
    if (source[i] !== '{') continue;
    const end = expressionEnd(source, i, 0);
    spans.push({ from: i, ...end });
    i = end.to - 1;
  }
  return spans;
}

export interface ScriptToken {
  from: number;
  to: number;
  text: string;
  kind: string;
  closed?: boolean;
}
const tokenPattern =
  /\/\/[^\n]*|(?:\d+)?[dD](?:\d+)?(?:(?:kh|kl|dh|dl)\d*)?(?![$_\p{L}\p{N}])|\d+(?:\.\d+)?|[$_\p{L}][$_\p{L}\p{N}:]*|\?\?|==|!=|<=|>=|&&|\|\||\*\*|\.\.|[^\s]/uy;
export function scriptTokens(source: string, offset = 0): ScriptToken[] {
  const tokens: ScriptToken[] = [];
  for (let index = 0; index < source.length; ) {
    if (/\s/.test(source[index])) {
      index++;
      continue;
    }
    if (source[index] === '"' || source[index] === "'" || source[index] === '`') {
      const end = quotedEnd(source, index, 0);
      tokens.push({
        from: offset + index,
        to: offset + end.to,
        text: source.slice(index, end.to),
        kind: 'string',
        closed: end.closed,
      });
      index = end.to;
      continue;
    }
    tokenPattern.lastIndex = index;
    const match = tokenPattern.exec(source)!;
    const text = match[0];
    const kind = text.startsWith('//')
      ? 'comment'
      : /^(?:\d+)?[dD](?:\d+)?(?:(?:kh|kl|dh|dl)\d*)?$/.test(text)
        ? 'dice'
        : /^\d/.test(text)
          ? 'number'
          : /^(if|else|while|func|return|break|continue|true|false|null)$/.test(text)
            ? 'keyword'
            : /^[$_\p{L}]/u.test(text)
              ? 'variable'
              : 'operator';
    tokens.push({ from: offset + index, to: offset + index + text.length, text, kind });
    index += text.length;
  }
  return tokens;
}
