import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
} from './types';
import {
  normalizeStoryPainterPlainMessage,
  renderForumText,
  renderPineappleForumText,
  renderTrgText,
} from './renderers';

export async function collectStoryPainterForumText(options: {
  chunks: AsyncIterable<StoryPainterLogItem[]>;
  chars: StoryPainterChar[];
  exportOptions: StoryPainterOptions;
  forumOptions: StoryPainterForumOptions;
  pineapple: boolean;
  colorByItem: (item: StoryPainterLogItem) => string;
}): Promise<string> {
  const lines: string[] = [];
  if (options.pineapple) {
    let previous: StoryPainterLogItem | undefined;
    let current: StoryPainterLogItem | undefined;
    const render = (next?: StoryPainterLogItem) =>
      renderPineappleForumText(
        current!,
        options.chars,
        options.exportOptions,
        options.forumOptions,
        options.colorByItem(current!),
        previous,
        next
      );
    for await (const chunk of options.chunks) {
      for (const item of chunk) {
        if (
          !normalizeStoryPainterPlainMessage(item, options.chars, {
            ...options.exportOptions,
            imageHide: true,
          }).trim()
        )
          continue;
        if (current) lines.push(render(item));
        previous = current;
        current = item;
      }
    }
    if (current) lines.push(render());
    return lines.join('\n');
  }

  for await (const chunk of options.chunks) {
    lines.push(
      ...chunk.map(item =>
        renderForumText(
          item,
          options.chars,
          options.exportOptions,
          options.forumOptions,
          options.colorByItem(item)
        )
      )
    );
  }
  return lines.join('\n');
}

export async function collectStoryPainterTrgText(options: {
  chunks: AsyncIterable<StoryPainterLogItem[]>;
  chars: StoryPainterChar[];
  exportOptions: StoryPainterOptions;
  addVoiceMark: boolean;
}): Promise<string> {
  const lines: string[] = [];
  for await (const chunk of options.chunks) {
    lines.push(
      ...chunk.map(item =>
        renderTrgText(item, options.chars, options.exportOptions, options.addVoiceMark)
      )
    );
  }
  return lines.join('\n');
}
