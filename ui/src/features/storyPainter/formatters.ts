import dayjs from 'dayjs';
import type { StoryPainterChar, StoryPainterLogItem, StoryPainterOptions } from './types';
import { replaceAllText } from './string';

const imageStyle = 'max-width: 300px';

export function escapeStoryPainterHtml(html: string): string {
  return html
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function formatStoryPainterImages(message: string, options: StoryPainterOptions, htmlText = false): string {
  let msg = message;
  if (options.imageHide) {
    msg = msg.replace(/\[CQ:(image|face)(,summary=\[动画表情\])?,[^\]]+\]/g, '');
  } else if (htmlText) {
    msg = msg.replace(
      /\[CQ:image(,summary=\[动画表情\])?,[^\]]+?file_unique=([a-zA-Z0-9]{32})\]/g,
      `<img style="${imageStyle}" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-$2/0?term=2" crossorigin="anonymous" />`,
    );
    msg = msg.replace(
      /\[CQ:image,summary=\[动画表情\],[^\]]+?url=([^\]]+)\]/g,
      `<img style="${imageStyle}" src="$1" />`,
    );
    msg = msg.replace(/\[CQ:image,[^\]]+?url=([^\]]+)\]/g, `<img style="${imageStyle}" src="$1" />`);
    msg = msg.replace(/\[CQ:image,file=(https?:\/\/[^\]]+)\]/g, `<img style="${imageStyle}" src="$1" />`);
    msg = msg.replace(
      /\[CQ:image,file=([A-Za-z0-9]{32,64})(\.[a-zA-Z]+?)\]/g,
      `<img style="${imageStyle}" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-$1/0?term=2,subType=1" />`,
    );
    msg = msg.replace(/\[CQ:image,file=file:\/\/[^\]]+([A-Za-z0-9]{32})(\.[a-zA-Z]+?)\]/g, (_m: string, p1: string) => {
      return `<img style="${imageStyle}" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-${String(p1).toUpperCase()}/0?term=2,subType=1" />`;
    });
    msg = msg.replace(
      /\[CQ:image,file=\{([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)}([^\]]+?)\]/g,
      `<img style="${imageStyle}" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-$1$2$3$4$5/0?term=2" />`,
    );
  }

  if (options.imageHide) {
    msg = msg.replace(/\[mirai:(image|marketface):[^\]]+\]/g, '');
  } else if (htmlText) {
    msg = msg.replace(
      /\[mirai:image:\{([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)}([^\]]+?)\]/g,
      `<img style="${imageStyle}" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-$1$2$3$4$5/0?term=2" />`,
    );
  }

  if (options.imageHide) {
    msg = msg.replace(/\[(image|图):[^\]]+\]/g, '');
  } else if (htmlText) {
    msg = msg.replace(/\[(?:image|图):([^\]]+)?([^\]]+)\]/g, `<img style="${imageStyle}" src="$1" />`);
  }

  return msg;
}

export function formatStoryPainterOffTopic(message: string, options: StoryPainterOptions, isDice = false): string {
  if (!options.offTopicHide || isDice) return message;
  return message.replace(/^\s*(?:@\S+\s+)*[（\(].*/gm, '');
}

export function formatStoryPainterCommands(message: string, options: StoryPainterOptions): string {
  if (!options.commandHide) return message;
  return message.replace(/^[\.。\/](?![\.。\/])(.|\n)*$/g, '');
}

export function finishStoryPainterMessage(message: string, options: StoryPainterOptions, isDice = false): string {
  let msg = message;
  if (isDice) {
    msg = replaceAllText(replaceAllText(msg, '<', ''), '>', '');
  }
  msg = msg.replace(/\[CQ:(?!image|at).+?,[^\]]+\]/g, '');
  msg = msg.replace(/\[mirai:(?!image).+?:[^\]]+\]/g, '');
  return msg.trim();
}

export function formatStoryPainterAt(message: string, pcList: StoryPainterChar[]): string {
  let msg = message;
  msg = msg.replace(/((\[CQ:at,[^\]]*qq=([0-9]+)[^\]]*\])) \[CQ:at,[^\]]*qq=\3[^\]]*\]/g, '$1');

  const pcMap = new Map<string, string>();
  pcList.forEach((pc) => {
    if (pc.IMUserId) pcMap.set(String(pc.IMUserId), pc.name);
  });

  const decodeHtml = (text: string) =>
    text
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&amp;/g, '&')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'");

  msg = msg.replace(/\[CQ:at,([^\]]+)\]/g, (_match, attrStr) => {
    const attrMap = new Map<string, string>();
    String(attrStr)
      .split(',')
      .forEach((part) => {
        const [key, ...rest] = part.split('=');
        if (key) attrMap.set(key.trim(), rest.join('=') || '');
      });

    const qq = attrMap.get('qq') || '';
    const mappedName = pcMap.get(qq);
    if (mappedName) return `@${mappedName}`;

    const nameAttr = attrMap.get('name');
    if (nameAttr) {
      const decoded = decodeHtml(nameAttr);
      return decoded.startsWith('@') ? decoded : `@${decoded}`;
    }
    if (qq) return `@${qq}`;
    return '@未知用户';
  });

  pcList.forEach((pc) => {
    msg = msg.replace(new RegExp(`&lt;@${escapeRegExp(pc.IMUserId)}&gt;`, 'g'), `@${pc.name}`);
    msg = msg.replace(new RegExp(`\\(met\\)${escapeRegExp(pc.IMUserId)}\\(met\\)`, 'g'), `@${pc.name}`);
  });

  return msg;
}

export function normalizeStoryPainterMessage(
  item: StoryPainterLogItem,
  pcList: StoryPainterChar[],
  options: StoryPainterOptions,
  htmlText = true,
): string {
  let msg = formatStoryPainterImages(escapeStoryPainterHtml(item.message), options, htmlText);
  msg = formatStoryPainterAt(msg, pcList);
  msg = formatStoryPainterOffTopic(msg, options, item.isDice);
  msg = formatStoryPainterCommands(msg, options);
  msg = finishStoryPainterMessage(msg, options, item.isDice);
  return formatStoryPainterOffTopic(msg, options, item.isDice);
}

export function storyPainterNickname(item: StoryPainterLogItem, options: StoryPainterOptions, trailingColon = true): string {
  const userId = options.userIdHide ? '' : `(${item.IMUserId})`;
  return `<${item.nickname}${userId}>${trailingColon ? ':' : ''}`;
}

export function storyPainterTime(item: StoryPainterLogItem, options: StoryPainterOptions): string {
  if (options.timeHide) return '';
  if (typeof item.time === 'number' && item.time !== 0) {
    return dayjs.unix(item.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss');
  }
  if (item.timeText) return item.timeText;
  return dayjs.unix(item.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss');
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
