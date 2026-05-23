import type { StoryPainterChar, StoryPainterForumOptions, StoryPainterLogItem, StoryPainterOptions } from './types';
import { storyPainterNickname } from './formatters';
import { colorHexToForumName, normalizeStoryPainterPlainMessage, renderForumText, renderTrgText } from './renderers';

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
    let current: PineappleBlock | null = null;
    for await (const chunk of options.chunks) {
      chunk.forEach((item) => {
        current = pushPineappleItem(current, item, options, lines);
      });
    }
    if (current) lines.push(formatPineappleBlock(current));
    return lines.join('\n');
  }

  for await (const chunk of options.chunks) {
    lines.push(...chunk.map((item) =>
      renderForumText(item, options.chars, options.exportOptions, options.forumOptions, options.colorByItem(item)),
    ));
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
    lines.push(...chunk.map((item) =>
      renderTrgText(item, options.chars, options.exportOptions, options.addVoiceMark),
    ));
  }
  return lines.join('\n');
}

type PineappleBlock = {
  key: string;
  name: string;
  color: string;
  lines: string[];
};

function pushPineappleItem(
  current: PineappleBlock | null,
  item: StoryPainterLogItem,
  options: {
    chars: StoryPainterChar[];
    exportOptions: StoryPainterOptions;
    forumOptions: StoryPainterForumOptions;
    colorByItem: (item: StoryPainterLogItem) => string;
  },
  output: string[],
): PineappleBlock | null {
  const text = normalizeStoryPainterPlainMessage(item, options.chars, { ...options.exportOptions, imageHide: true });
  if (!text) return current;

  const key = `${item.nickname}-${item.IMUserId}`;
  if (current && current.key === key) {
    current.lines.push(text);
    return current;
  }

  if (current) output.push(formatPineappleBlock(current));
  return {
    key,
    name: storyPainterNickname(item, options.exportOptions, false),
    color: options.forumOptions.bbsUseColorName ? colorHexToForumName(options.colorByItem(item)) : options.colorByItem(item),
    lines: [text],
  };
}

function formatPineappleBlock(block: PineappleBlock): string {
  return `[color=silver]${block.name}[/color][color=${block.color}] ${block.lines.join('\n')} [/color]`;
}
