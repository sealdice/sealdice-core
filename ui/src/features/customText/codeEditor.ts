import { basicSetup } from 'codemirror';
import { EditorState, StateField } from '@codemirror/state';
import { Decoration, EditorView } from '@codemirror/view';
import { markdown } from '@codemirror/lang-markdown';
import { HighlightStyle, defaultHighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { templateHighlights } from './highlight';

export interface EditorSnapshot {
  anchor: number;
  head: number;
  scrollTop: number;
}
export interface ActiveTextEditor {
  snapshot(): EditorSnapshot;
  setText(value: string): void;
  destroy(): void;
}

const marks = (state: EditorState) =>
  Decoration.set(
    templateHighlights(state.doc.toString()).map(span =>
      Decoration.mark({ class: `seal-${span.kind}` }).range(span.from, span.to)
    ),
    true
  );
const highlights = StateField.define({
  create: marks,
  update: (value, transaction) => (transaction.docChanged ? marks(transaction.state) : value),
  provide: field => EditorView.decorations.from(field),
});
// Keep Markdown styling within the same UI palette; script marks use the shared theme CSS.
const markdownStyle = HighlightStyle.define(
  defaultHighlightStyle.specs.map(spec => ({
    ...spec,
    color: 'var(--sd-text-secondary)',
  }))
);

export function createTextEditor(
  parent: HTMLElement,
  text: string,
  selection: EditorSnapshot,
  onChange: (text: string) => void
): ActiveTextEditor {
  const view = new EditorView({
    parent,
    state: EditorState.create({
      doc: text,
      selection: {
        anchor: Math.min(selection.anchor, text.length),
        head: Math.min(selection.head, text.length),
      },
      extensions: [
        basicSetup,
        markdown(),
        syntaxHighlighting(markdownStyle),
        highlights,
        EditorView.lineWrapping,
        EditorView.contentAttributes.of({
          role: 'textbox',
          'aria-multiline': 'true',
          'aria-label': '文案代码编辑器',
          spellcheck: 'false',
        }),
        EditorView.updateListener.of(update => {
          if (update.docChanged) onChange(update.state.doc.toString());
        }),
      ],
    }),
  });
  view.focus();
  view.requestMeasure({
    read: () => undefined,
    write: () => {
      view.scrollDOM.scrollTop = selection.scrollTop;
    },
  });
  return {
    snapshot: () => ({
      anchor: view.state.selection.main.anchor,
      head: view.state.selection.main.head,
      scrollTop: view.scrollDOM.scrollTop,
    }),
    setText(value) {
      if (view.state.doc.toString() !== value)
        view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value } });
    },
    destroy: () => view.destroy(),
  };
}
