import type { StoryPainterChar, StoryPainterLogItem, StoryPainterOptions, StoryPainterRole } from './types';
import { packStoryPainterNameId } from './types';
import { replaceAllText } from './string';

export const storyPainterPalette = [
  '#db2777',
  '#ea580c',
  '#f472b6',
  '#c084fc',
  '#0284c7',
  '#94a3b8',
  '#4b5563',
];

export function inferStoryPainterRole(item: StoryPainterLogItem): StoryPainterRole {
  if (item.role) return item.role;
  if (item.isDice) return '骰子';
  if (item.nickname.toLowerCase().startsWith('ob')) return '隐藏';
  return '角色';
}

export function buildStoryPainterChars(
  items: StoryPainterLogItem[],
  savedNameColors = new Map<string, string>(),
): StoryPainterChar[] {
  const chars: StoryPainterChar[] = [];
  const seen = new Set<string>();
  let colorIndex = 0;

  items.forEach((item) => {
    if (item.isRaw) return;
    const role = inferStoryPainterRole(item);
    const candidate: StoryPainterChar = {
      name: item.nickname,
      IMUserId: item.IMUserId,
      role,
      color: savedNameColors.get(item.nickname) || storyPainterPalette[colorIndex % storyPainterPalette.length],
    };
    const id = packStoryPainterNameId(candidate);
    if (seen.has(id)) return;
    seen.add(id);
    if (!savedNameColors.has(item.nickname)) colorIndex += 1;
    chars.push(candidate);
  });

  return chars;
}

export function buildStoryPainterPcMap(chars: StoryPainterChar[]): Map<string, StoryPainterChar> {
  const pcMap = new Map<string, StoryPainterChar>();
  chars.forEach((pc) => pcMap.set(packStoryPainterNameId(pc), pc));
  return pcMap;
}

export function isStoryPainterHidden(item: StoryPainterLogItem, chars: StoryPainterChar[]): boolean {
  if (item.role === '隐藏') return true;
  const pc = buildStoryPainterPcMap(chars).get(packStoryPainterNameId(item));
  if (pc?.role === '隐藏') return true;
  return chars.some((char) => char.role === '隐藏' && char.IMUserId === item.IMUserId && char.name === item.nickname);
}

export function buildStoryPainterPreviewItems(
  items: StoryPainterLogItem[],
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
  normalizeMessage: (item: StoryPainterLogItem) => string,
): StoryPainterLogItem[] {
  const previewItems: StoryPainterLogItem[] = [];
  items.forEach((item) => {
    if (item.isRaw) return;
    if (isStoryPainterHidden(item, chars)) return;
    const msg = normalizeMessage(item);
    if (msg.trim() === '') return;
    previewItems.push({ ...item, index: item.index ?? previewItems.length });
  });
  void options;
  return previewItems;
}

export function renameStoryPainterChar(
  items: StoryPainterLogItem[],
  target: StoryPainterChar,
  nextName: string,
): StoryPainterLogItem[] {
  return items.map((item) => {
    let message = replaceAllText(item.message, `<${target.name}>`, `<${nextName}>`);
    if (item.IMUserId === target.IMUserId && item.nickname === target.name) {
      return { ...item, nickname: nextName, message };
    }
    return { ...item, message };
  });
}

export function deleteStoryPainterChar(items: StoryPainterLogItem[], target: StoryPainterChar): StoryPainterLogItem[] {
  return items.filter((item) => !(item.IMUserId === target.IMUserId && item.nickname === target.name));
}
