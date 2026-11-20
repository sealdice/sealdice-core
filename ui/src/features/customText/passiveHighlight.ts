import { templateHighlights } from './highlight';

export interface HighlightPart {
  text: string;
  classes: string;
}

/** Flatten overlapping marks into escaped Vue text nodes, without an editor/parser instance. */
export function passiveHighlights(source: string): HighlightPart[] {
  const events = templateHighlights(source)
    .flatMap(span => [
      { position: span.from, kind: span.kind, delta: 1 },
      { position: span.to, kind: span.kind, delta: -1 },
    ])
    .sort((a, b) => a.position - b.position);
  const parts: HighlightPart[] = [];
  const active = new Map<string, number>();
  let cursor = 0;
  for (const event of events) {
    if (event.position > cursor) {
      parts.push({
        text: source.slice(cursor, event.position),
        classes: [...active.keys()].map(kind => `seal-${kind}`).join(' '),
      });
      cursor = event.position;
    }
    const count = (active.get(event.kind) ?? 0) + event.delta;
    if (count) active.set(event.kind, count);
    else active.delete(event.kind);
  }
  if (cursor < source.length) parts.push({ text: source.slice(cursor), classes: '' });
  return parts;
}
