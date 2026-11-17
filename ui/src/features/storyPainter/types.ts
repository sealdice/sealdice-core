export type StoryPainterRole = '主持人' | '角色' | '骰子' | '隐藏';
export type StoryPainterPreviewDisplayMode = 'preview' | 'bbs' | 'bbspineapple' | 'trg';

export interface StoryPainterChar {
  name: string;
  IMUserId: string;
  role: StoryPainterRole;
  color: string;
}

export interface StoryPainterLogItem {
  id: number;
  nickname: string;
  IMUserId: string;
  time: number;
  timeText?: string;
  message: string;
  isDice: boolean;
  commandId: number;
  commandInfo?: unknown;
  uniformId?: string;
  color?: string;
  role?: StoryPainterRole;
  isRaw?: boolean;
  index?: number;
  version?: number;
}

export interface StoryPainterOptions {
  commandHide: boolean;
  imageHide: boolean;
  offTopicHide: boolean;
  timeHide: boolean;
  userIdHide: boolean;
  yearHide: boolean;
  textIndentAll: boolean;
  textIndentFirst: boolean;
}

export interface StoryPainterForumOptions {
  bbsUseSpaceWithMultiLine: boolean;
  bbsUseColorName: boolean;
}

export function packStoryPainterNameId(item: StoryPainterChar | StoryPainterLogItem): string {
  const name = 'name' in item ? item.name : item.nickname;
  return `${name}-${item.IMUserId}`;
}

export function defaultStoryPainterOptions(): StoryPainterOptions {
  return {
    commandHide: false,
    imageHide: false,
    offTopicHide: false,
    timeHide: false,
    userIdHide: true,
    yearHide: true,
    textIndentAll: false,
    textIndentFirst: true,
  };
}
