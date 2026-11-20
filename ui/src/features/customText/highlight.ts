import { scriptSpans, scriptTokens } from './syntax';

export interface TemplateHighlight {
  from: number;
  to: number;
  kind: string;
}

/** Split string-colour ranges around interpolations instead of overlapping conflicting text colours. */
export function templateHighlights(source: string): TemplateHighlight[] {
  const highlights: TemplateHighlight[] = [];
  const add = (from: number, to: number, kind: string) => {
    if (to > from) highlights.push({ from, to, kind });
  };

  function string(from: number, to: number) {
    const text = source.slice(from, to);
    // Keep unknown sequences such as TeX's \boxed literal instead of treating every backslash as an escape.
    const escapes = /\\[nrtf\\"'`{}]/g;
    let cursor = from;
    for (const match of text.matchAll(escapes)) {
      const start = from + match.index!;
      add(cursor, start, 'string');
      add(start, start + match[0].length, 'escape');
      cursor = start + match[0].length;
    }
    add(cursor, to, 'string');
  }

  function expressions(text: string, offset: number, nested: boolean, depth: number) {
    for (const span of scriptSpans(text)) {
      const block = text.startsWith('{%', span.from);
      const start = span.from + (block ? 2 : 1);
      const blockClosed = block && span.closed && text.slice(span.to - 2, span.to) === '%}';
      const end = span.closed ? span.to - (blockClosed ? 2 : 1) : span.to;
      if (block) {
        add(offset + span.from, offset + start, 'expression');
        if (blockClosed) add(offset + end, offset + span.to, 'expression');
      } else {
        add(offset + span.from, offset + span.to, nested ? 'interpolation' : 'expression');
      }
      add(offset + span.from, offset + start, 'delimiter');
      code(text.slice(start, end), offset + start, depth);
      if (span.closed) add(offset + end, offset + span.to, 'delimiter');
    }
  }

  function code(text: string, offset: number, depth: number) {
    for (const token of scriptTokens(text, offset)) {
      if (token.kind !== 'string' || !token.text.startsWith('`') || depth >= 32) {
        if (token.kind === 'string') string(token.from, token.to);
        else add(token.from, token.to, token.kind);
        continue;
      }
      const end = token.closed ? token.text.length - 1 : token.text.length;
      const body = token.text.slice(1, end);
      let cursor = token.from;
      for (const span of scriptSpans(body)) {
        const start = token.from + 1 + span.from;
        string(cursor, start);
        expressions(body.slice(span.from, span.to), start, true, depth + 1);
        cursor = token.from + 1 + span.to;
      }
      string(cursor, token.to);
    }
  }

  expressions(source, 0, false, 0);
  return highlights;
}
