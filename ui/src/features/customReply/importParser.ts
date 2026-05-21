import type { ReplyTask } from './model';

export type ReplyImportLine = {
  conditions: string[];
  replies: string[];
  rest: string;
};

export function parseReplyImportLine(input: string): ReplyImportLine {
  const conditions: string[] = [];
  const replies: string[] = [];
  let restIndex = 0;

  let currentStr = '';
  let isLeft = true;
  let isEscaped = false;

  for (let i = 0; i < input.length; i++) {
    const char = input[i];
    restIndex = i;
    if (isEscaped) {
      if (char !== '\r' && char !== '\n' && char !== '/') {
        currentStr += '\\';
      }
      if (char === 'n' || char === 'r') {
        currentStr = `${currentStr.slice(0, -1)}\n`;
      } else {
        currentStr += char;
      }
      isEscaped = false;
      continue;
    }
    if (char === '\n') break;
    if (char === '\\') {
      isEscaped = true;
      continue;
    }
    if (char === '|') {
      if (isLeft) conditions.push(currentStr);
      else replies.push(currentStr);
      currentStr = '';
    } else if (char === '/') {
      if (i < input.length - 1 && input[i + 1] === '|') {
        currentStr += char;
      } else {
        if (isLeft) conditions.push(currentStr);
        else replies.push(currentStr);
        currentStr = '';
        isLeft = false;
      }
    } else {
      currentStr += char;
    }
  }
  if (isLeft) conditions.push(currentStr);
  else replies.push(currentStr);
  return {
    conditions,
    replies,
    rest: input.slice(restIndex + 1),
  };
}

export function parseReplyImportText(input: string): ReplyTask[] {
  const tasks: ReplyTask[] = [];
  let text = input;
  while (text) {
    const { conditions, replies, rest } = parseReplyImportLine(text);
    if (conditions.length && replies.length) {
      tasks.push({
        enable: true,
        conditions: [{
          condType: 'textMatch',
          matchType: 'matchMulti',
          value: conditions.join('|'),
        }],
        results: [{
          resultType: 'replyToSender',
          delay: 0,
          message: replies.map(reply => [reply, 1]),
        }],
      });
    }
    text = rest;
  }
  return tasks;
}
