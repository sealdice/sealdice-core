import dayjs from 'dayjs';
import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
} from './types';
import { escapeStoryPainterHtml, normalizeStoryPainterMessage, storyPainterNickname, storyPainterTime } from './formatters';
import { replaceAllText } from './string';

const timeColor = '#9ca3af';
const fallbackColor = '#4b5563';
const bbsColorNames = ['skyblue', 'royalblue', 'darkblue', 'orangered', 'red', 'firebrick', 'darkred', 'green', 'limegreen', 'seagreen', 'tomato', 'coral', 'indigo', 'burlywood', 'sandybrown', 'chocolate'];

export function renderPreviewHtml(
  item: StoryPainterLogItem,
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
  color: string,
): string {
  const time = storyPainterTime(item, options);
  const nickname = storyPainterNickname(item, options);
  let message = normalizeStoryPainterMessage(item, chars, options, true);
  if (item.isDice) {
    message = replaceCharacterBrackets(message, chars);
  }
  return [
    time ? `<span class="_time" style="color:${timeColor}">${escapeStoryPainterHtml(time)}</span>` : '',
    `<span class="_nickname" style="color:${color}">${escapeStoryPainterHtml(nickname)}</span>`,
    `<span class="_message" style="color:${color}">${replaceAllText(message, '\n', '<br />')}</span>`,
  ].filter(Boolean).join(' ');
}

export function renderForumText(
  item: StoryPainterLogItem,
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
  forumOptions: StoryPainterForumOptions,
  color: string,
): string {
  const bbsOptions = { ...options, imageHide: true };
  const displayColor = forumOptions.bbsUseColorName ? colorHexToForumName(color) : color || fallbackColor;
  const time = storyPainterTime(item, options);
  const timeText = forumOptions.bbsUseColorName ? 'silver' : timeColor;
  const nickname = storyPainterNickname(item, options, false);
  const message = normalizeStoryPainterPlainMessage(item, chars, bbsOptions);
  const prefix = `${time ? `[color=${timeText}]${time}[/color] ` : ''}[color=${displayColor}]${nickname} `;
  const lines = message.trim().split(/\r?\n/);
  if (forumOptions.bbsUseSpaceWithMultiLine) {
    const continuation = `${time ? spaceLike(time) : '\u2002'}${nickname}`;
    const text = lines.map((line, index) => index === 0 ? line : `\n${continuation}${line}`).join('');
    return `${prefix}${text}[/color]`;
  }
  const continuation = `${time ? `[color=${timeText}]${time}[/color]` : ''}[color=${displayColor}] ${nickname}`;
  const text = lines.map((line, index) => index === 0 ? line : `[/color]\n${continuation}${line}`).join('');
  return `${prefix}${text}[/color]`;
}

export function normalizeStoryPainterPlainMessage(
  item: StoryPainterLogItem,
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
): string {
  let message = normalizeStoryPainterMessage(item, chars, options, false);
  if (item.isDice) {
    message = replaceCharacterBrackets(message, chars);
  }
  return decodeStoryPainterHtml(message).replace(/<br\s*\/?>/gi, '\n');
}

export function renderPineappleForumBlocks(
  items: StoryPainterLogItem[],
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
  forumOptions: StoryPainterForumOptions,
  colorByItem: (item: StoryPainterLogItem) => string,
): string[] {
  const blocks: Array<{ key: string; name: string; color: string; lines: string[] }> = [];
  let current: { key: string; name: string; color: string; lines: string[] } | null = null;

  items.forEach((item) => {
    const text = normalizeStoryPainterPlainMessage(item, chars, { ...options, imageHide: true });
    if (!text) return;
    const key = `${item.nickname}-${item.IMUserId}`;
    if (current && current.key === key) {
      current.lines.push(text);
      return;
    }
    if (current) blocks.push(current);
    current = {
      key,
      name: storyPainterNickname(item, options, false),
      color: forumOptions.bbsUseColorName ? colorHexToForumName(colorByItem(item)) : colorByItem(item),
      lines: [text],
    };
  });
  if (current) blocks.push(current);

  return blocks.map((block) => `[color=silver]${block.name}[/color][color=${block.color}] ${block.lines.join('\n')} [/color]`);
}

export function renderTrgText(
  item: StoryPainterLogItem,
  chars: StoryPainterChar[],
  options: StoryPainterOptions,
  addVoiceMark: boolean,
): string {
  let message = normalizeStoryPainterPlainMessage(item, chars, options);
  message = replaceAllText(replaceAllText(message.trim(), '"', ''), '\\', '');
  const pc = chars.find((char) => char.name === item.nickname);
  const kpFlag = pc?.role === '主持人' ? ',KP' : '';
  const speaker = `[${item.nickname}${kpFlag}]:`;
  const prefix = item.isDice ? '# ' : '';
  const voice = addVoiceMark && !item.isDice ? '{*}' : '';
  const text = message.split(/\r?\n/).map((line) => `${prefix}${speaker}${line}${voice}`).join('\n');
  const commandText = renderTrgCommandText(item.commandInfo);
  return commandText ? `${text}\n${commandText}` : text;
}

export function renderRawText(items: StoryPainterLogItem[]): string {
  return items.map((item) => {
    const time = item.time ? dayjs.unix(item.time).format('YYYY/MM/DD HH:mm:ss') : '';
    return `${item.nickname}(${item.IMUserId}) ${time}\n${item.message}\n`;
  }).join('\n');
}

export function colorHexToForumName(color: string): string {
  switch (color.toLowerCase()) {
    case '#d97706':
      return 'sienna';
    case '#db2777':
      return 'crimson';
    case '#ea580c':
      return 'orange';
    case '#f472b6':
      return 'deeppink';
    case '#c084fc':
      return 'purple';
    case '#0284c7':
      return 'blue';
    case '#94a3b8':
      return 'teal';
    case '#4b5563':
    case '#9ca3af':
      return 'silver';
  }
  let hash = 0;
  for (let index = 0; index < color.length; index += 1) hash = (hash * 31 + color.charCodeAt(index)) >>> 0;
  return bbsColorNames[hash % bbsColorNames.length] || 'red';
}

function spaceLike(text: string): string {
  const parts: string[] = [];
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index];
    parts.push(char === ':' || char === '/' || char === '[' || char === ']' ? '\u00a0' : '\u2002');
  }
  parts.push('\u00a0');
  return parts.join('');
}

function replaceCharacterBrackets(message: string, chars: StoryPainterChar[]): string {
  let next = message;
  chars.forEach((char) => {
    next = replaceAllText(next, `&lt;${escapeStoryPainterHtml(char.name)}&gt;`, escapeStoryPainterHtml(char.name));
    next = replaceAllText(next, `<${char.name}>`, char.name);
  });
  return next;
}

function decodeStoryPainterHtml(value: string): string {
  return value
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&amp;/g, '&');
}

function renderTrgCommandText(commandInfo: unknown): string {
  const info = asRecord(commandInfo);
  if (!info) return '';
  const cmd = stringValue(info.cmd);
  const rule = stringValue(info.rule);
  const pcName = stringValue(info.pcName);
  const items = arrayValue(info.items);

  if (rule === 'coc7') {
    switch (cmd) {
      case 'ra':
        return items.map((item) => {
          const version = numberValue(item.version, 0);
          const diceNum = readDiceNum(stringValue(item.expr1));
          if (version === 101) {
            return `(${pcName}的${stringValue(item.expr2)},${diceNum},${stringValue(item.checkVal)},${stringValue(item.outcome)})`;
          }
          return `(${pcName}的${stringValue(item.expr2)},${diceNum},${stringValue(item.attrVal)},${stringValue(item.checkVal)})`;
        }).join(',');
      case 'st': {
        const hpItems = items
          .filter(item => stringValue(item.attr) === 'hp')
          .map((item) => `<hitpoint>:(${pcName},${Math.max(numberValue(item.valOld, 0), numberValue(item.valNew, 0))},${stringValue(item.valOld)},${stringValue(item.valNew)})`);
        return `# 请注意，当前版本需要手动调整下方最大生命值(第二项)\n${hpItems.join('\n')}`;
      }
      case 'sc':
        return `<dice>:${items.map((item) => {
          const exprs = arrayValue(item.exprs).map(value => String(value));
          const expr = exprs[0] ?? '';
          return `(${pcName}的${expr},${readDiceNum(expr)},${stringValue(item.sanOld)},${stringValue(item.outcome) || stringValue(item.checkVal)})`;
        }).join(',')}`;
    }
  }

  if (rule === 'dnd5e') {
    switch (cmd) {
      case 'st':
        return items
          .filter(item => stringValue(item.attr) === 'hp')
          .map((item) => `<hitpoint>:(${pcName},${Math.max(numberValue(item.valOld, 0), numberValue(item.valNew, 0))},${stringValue(item.valOld)},${stringValue(item.valNew)})`)
          .join('\n');
      case 'rc': {
        const text = items.map((item) => {
          const expr = stringValue(item.expr);
          return `(${pcName}的${stringValue(item.reason)}检定,${readDiceNum(expr, 20)},NA,${stringValue(item.result)})`;
        }).join(',');
        return `# 请注意，DND的最大面数可能为 D20+各种加值，需要手动二次调整\n<dice>:${text}`;
      }
    }
  }

  if (cmd === 'roll') {
    return `<dice>:${items.map((item) =>
      `(${pcName}的${stringValue(item.expr)},${readDiceNum(stringValue(item.expr))},NA,${stringValue(item.result)})`,
    ).join(',')}`;
  }

  return '';
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? value as Record<string, unknown> : null;
}

function arrayValue(value: unknown): Array<Record<string, unknown>> {
  return Array.isArray(value)
    ? value.map(asRecord).filter((item): item is Record<string, unknown> => Boolean(item))
    : [];
}

function stringValue(value: unknown): string {
  return value === undefined || value === null ? '' : String(value);
}

function numberValue(value: unknown, fallback: number): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function readDiceNum(expr: string, defaultVal = 100): number {
  const matched = /[dD](\d+)/.exec(expr);
  return matched?.[1] ? Number.parseInt(matched[1], 10) : defaultVal;
}
